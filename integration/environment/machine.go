package environment

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"code.cloudfoundry.org/quarks-statefulset/pkg/kube/client/clientset/versioned"
	"code.cloudfoundry.org/quarks-utils/testing/machine"
)

// Machine produces and destroys resources for tests
type Machine struct {
	machine.Machine

	VersionedClientset *versioned.Clientset
}

// CreateStatefulSet creates a statefulset and returns a function to delete it
func (m *Machine) CreateStatefulSet(namespace string, res appsv1.StatefulSet) (machine.TearDownFunc, error) {
	client := m.Clientset.AppsV1().StatefulSets(namespace)
	_, err := client.Create(context.Background(), &res, metav1.CreateOptions{})
	return func() error {
		err := client.Delete(context.Background(), res.GetName(), metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		return nil
	}, err
}
