package mutate

import (
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// StatefulSetMutateFn returns MutateFn which mutates StatefulSet including:
// - labels, annotations
// - spec
func StatefulSetMutateFn(sfs *appsv1.StatefulSet) controllerutil.MutateFn {
	updated := sfs.DeepCopy()
	return func() error {
		sfs.Labels = updated.Labels
		sfs.Annotations = updated.Annotations
		sfs.Spec.Replicas = updated.Spec.Replicas
		sfs.Spec.Template = updated.Spec.Template
		sfs.Spec.UpdateStrategy = updated.Spec.UpdateStrategy
		return nil
	}
}
