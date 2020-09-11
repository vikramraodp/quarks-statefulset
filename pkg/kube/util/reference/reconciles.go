package reference

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"code.cloudfoundry.org/quarks-statefulset/pkg/kube/apis"
	qstsv1a1 "code.cloudfoundry.org/quarks-statefulset/pkg/kube/apis/quarksstatefulset/v1alpha1"
	log "code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	vss "code.cloudfoundry.org/quarks-utils/pkg/versionedsecretstore"
)

// ReconcileType lists all the types of reconciliations we can return,
// for controllers that have types that can reference ConfigMaps or Secrets
type ReconcileType int

const (
	// ReconcileForQuarksStatefulSet represents the QuarksStatefulSet CRD
	ReconcileForQuarksStatefulSet = iota
	// ReconcileForPod represents the StatefulSet Kube Resource
)

func (r ReconcileType) String() string {
	return [...]string{
		"QuarksStatefulSet",
	}[r]
}

// GetReconciles returns reconciliation requests for QuarksStatefulSets
// that reference an object. The object can be a ConfigMap or a Secret
func GetReconciles(ctx context.Context, client crc.Client, object apis.Object, versionCheck bool) ([]reconcile.Request, error) {
	objReferencedBy := func(parent qstsv1a1.QuarksStatefulSet) (bool, error) {
		var (
			objectReferences map[string]bool
			name             string
			versionedSecret  bool
		)

		switch object := object.(type) {
		case *corev1.ConfigMap:
			objectReferences = getConfMapRefFromPod(parent.Spec.Template.Spec.Template.Spec)
		case *corev1.Secret:
			objectReferences = getSecretRefFromPodSpec(parent.Spec.Template.Spec.Template.Spec)
			versionedSecret = vss.IsVersionedSecret(*object)
		default:
			return false, errors.New("can't get reconciles for unknown object type; supported types are ConfigMap and Secret")
		}
		name = object.GetName()

		if versionedSecret {
			keys := make([]string, len(objectReferences))
			i := 0
			for k := range objectReferences {
				keys[i] = k
				i++
			}
			ok := vss.ContainsSecretName(keys, name)
			if versionCheck && ok {
				ok := vss.ContainsOutdatedSecretVersion(keys, name)
				return ok, nil
			}
			return ok, nil
		}

		_, ok := objectReferences[name]
		return ok, nil
	}

	namespace := object.GetNamespace()
	result := []reconcile.Request{}

	log.Debugf(ctx, "Searching 'qsts' for references to '%s/%s'", namespace, object.GetName())
	list := &qstsv1a1.QuarksStatefulSetList{}
	err := client.List(ctx, list, crc.InNamespace(namespace))
	if err != nil {
		return nil, errors.Wrap(err, "failed to list QuarksStatefulSets for ConfigMap reconciles")
	}

	for _, quarksStatefulSet := range list.Items {
		if !quarksStatefulSet.Spec.UpdateOnConfigChange {
			continue
		}

		isRef, err := objReferencedBy(quarksStatefulSet)
		if err != nil {
			return nil, err
		}

		if isRef {
			result = append(result, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      quarksStatefulSet.Name,
					Namespace: quarksStatefulSet.Namespace,
				}})
		}
	}

	return result, nil
}
