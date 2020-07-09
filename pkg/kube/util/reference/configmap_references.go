package reference

import (
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	qstsv1a1 "code.cloudfoundry.org/quarks-statefulset/pkg/kube/apis/quarksstatefulset/v1alpha1"
)

// GetConfigMapsReferencedBy returns a list of all names for ConfigMaps referenced by the object
// The object can be an QuarksStatefulSet or a BOSHDeployment
func GetConfigMapsReferencedBy(object interface{}) (map[string]bool, error) {
	// Figure out the type of object
	switch object := object.(type) {
	case qstsv1a1.QuarksStatefulSet:
		return getConfMapRefFromESts(object), nil
	default:
		return nil, errors.New("can't get config map references for unknown type; supported types are BOSHDeployment and QuarksStatefulSet")
	}
}

func getConfMapRefFromESts(object qstsv1a1.QuarksStatefulSet) map[string]bool {
	return getConfMapRefFromPod(object.Spec.Template.Spec.Template.Spec)
}

func getConfMapRefFromPod(object corev1.PodSpec) map[string]bool {
	result := map[string]bool{}

	// Look at all volumes
	for _, volume := range object.Volumes {
		if volume.VolumeSource.ConfigMap != nil {
			result[volume.VolumeSource.ConfigMap.Name] = true
		}
	}

	// Look at all init containers
	for _, container := range object.InitContainers {
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				result[envFrom.ConfigMapRef.Name] = true
			}
		}

		for _, envVar := range container.Env {
			if envVar.ValueFrom != nil && envVar.ValueFrom.ConfigMapKeyRef != nil {
				result[envVar.ValueFrom.ConfigMapKeyRef.Name] = true
			}
		}
	}

	// Look at all containers
	for _, container := range object.Containers {
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil {
				result[envFrom.ConfigMapRef.Name] = true
			}
		}

		for _, envVar := range container.Env {
			if envVar.ValueFrom != nil && envVar.ValueFrom.ConfigMapKeyRef != nil {
				result[envVar.ValueFrom.ConfigMapKeyRef.Name] = true
			}
		}
	}

	return result
}
