package quarksstatefulset

import (
	"context"
	"strconv"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	crc "sigs.k8s.io/controller-runtime/pkg/client"

	qstsv1a1 "code.cloudfoundry.org/quarks-statefulset/pkg/kube/apis/quarksstatefulset/v1alpha1"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
)

// GetMaxStatefulSetVersion returns the max version statefulSet
// of the quarksStatefulSet.
func GetMaxStatefulSetVersion(ctx context.Context, client crc.Client, qStatefulSet *qstsv1a1.QuarksStatefulSet) (*appsv1.StatefulSet, int, error) {
	// Default response is an empty StatefulSet with version '0' and an empty signature
	result := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				qstsv1a1.AnnotationVersion: "0",
			},
		},
	}
	maxVersion := 0

	if qStatefulSet.Namespace == "" {
		return result, maxVersion, nil
	}

	statefulSets, err := listStatefulSetsFromInformer(ctx, client, qStatefulSet)
	if err != nil {
		return nil, 0, err
	}

	for _, ss := range statefulSets {
		strVersion := ss.Annotations[qstsv1a1.AnnotationVersion]
		if strVersion == "" {
			return nil, 0, errors.Errorf("The statefulset '%s/%s' does not have the annotation(%s), a version could not be retrieved.", ss.Namespace, ss.Name, qstsv1a1.AnnotationVersion)
		}

		version, err := strconv.Atoi(strVersion)
		if err != nil {
			return nil, 0, err
		}

		if ss.Annotations != nil && version > maxVersion {
			result = &ss
			maxVersion = version
		}
	}

	ctxlog.Debugf(ctx, "Latest StatefulSet owned by QuarksStatefulSet '%s' has version '%d'", qStatefulSet.GetNamespacedName(), maxVersion)

	return result, maxVersion, nil
}

// listStatefulSetsFromInformer gets StatefulSets cross version owned by the QuarksStatefulSet from informer
func listStatefulSetsFromInformer(ctx context.Context, client crc.Client, qStatefulSet *qstsv1a1.QuarksStatefulSet) ([]appsv1.StatefulSet, error) {
	allStatefulSets := &appsv1.StatefulSetList{}
	err := client.List(ctx, allStatefulSets,
		crc.InNamespace(qStatefulSet.Namespace),
	)
	if err != nil {
		return nil, err
	}

	return filterByOwner(qStatefulSet, allStatefulSets.Items)
}

// listStatefulSetsFromAPIClient gets StatefulSets cross version owned by the QuarksStatefulSet from API client directly
func listStatefulSetsFromAPIClient(ctx context.Context, client appsv1client.AppsV1Interface, qStatefulSet *qstsv1a1.QuarksStatefulSet) ([]appsv1.StatefulSet, error) {
	allStatefulSets, err := client.StatefulSets(qStatefulSet.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return filterByOwner(qStatefulSet, allStatefulSets.Items)
}

// filterByOwner gets StatefulSets owned by the QuarksStatefulSet
func filterByOwner(qStatefulSet *qstsv1a1.QuarksStatefulSet, statefulSets []appsv1.StatefulSet) ([]appsv1.StatefulSet, error) {
	result := []appsv1.StatefulSet{}

	for _, statefulSet := range statefulSets {
		if metav1.IsControlledBy(&statefulSet, qStatefulSet) {
			result = append(result, statefulSet)
		}
	}

	return result, nil
}
