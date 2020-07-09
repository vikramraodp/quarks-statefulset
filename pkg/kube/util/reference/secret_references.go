package reference

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	crc "sigs.k8s.io/controller-runtime/pkg/client"

	qstsv1a1 "code.cloudfoundry.org/quarks-statefulset/pkg/kube/apis/quarksstatefulset/v1alpha1"
)

// GetSecretsReferencedBy returns a list of all names for Secrets referenced by the object
// The object can be an QuarksStatefulSet or a BOSHDeployment
func GetSecretsReferencedBy(ctx context.Context, client crc.Client, object interface{}) (map[string]bool, error) {
	switch object := object.(type) {
	case qstsv1a1.QuarksStatefulSet:
		return getSecretRefFromESts(object), nil
	case corev1.Pod:
		return getSecretRefFromPod(object), nil
	default:
		return nil, errors.New("can't get secret references for unknown type; supported types are BOSHDeployment and QuarksStatefulSet")
	}
}

func getSecretRefFromPod(object corev1.Pod) map[string]bool {
	return getSecretRefFromPodSpec(object.Spec)
}

func getSecretRefFromESts(object qstsv1a1.QuarksStatefulSet) map[string]bool {
	return getSecretRefFromPodSpec(object.Spec.Template.Spec.Template.Spec)
}

func getSecretRefFromPodSpec(object corev1.PodSpec) map[string]bool {
	result := map[string]bool{}

	// Look at all volumes
	for _, volume := range object.Volumes {
		if volume.VolumeSource.Secret != nil {
			result[volume.VolumeSource.Secret.SecretName] = true
		}
	}

	// Look at all init containers
	for _, container := range object.InitContainers {
		for _, envFrom := range container.EnvFrom {
			if envFrom.SecretRef != nil {
				result[envFrom.SecretRef.Name] = true
			}
		}

		for _, envVar := range container.Env {
			if envVar.ValueFrom != nil && envVar.ValueFrom.SecretKeyRef != nil {
				result[envVar.ValueFrom.SecretKeyRef.Name] = true
			}
		}
	}

	// Look at all containers
	for _, container := range object.Containers {
		for _, envFrom := range container.EnvFrom {
			if envFrom.SecretRef != nil {
				result[envFrom.SecretRef.Name] = true
			}
		}

		for _, envVar := range container.Env {
			if envVar.ValueFrom != nil && envVar.ValueFrom.SecretKeyRef != nil {
				result[envVar.ValueFrom.SecretKeyRef.Name] = true
			}
		}
	}

	return result
}
