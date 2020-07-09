package environment

import (
	"context"
	"sync/atomic"
	"time"

	gomegaConfig "github.com/onsi/ginkgo/config"
	"github.com/onsi/gomega"
	"github.com/spf13/afero"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" //from https://github.com/kubernetes/client-go/issues/345
	"k8s.io/client-go/rest"

	"code.cloudfoundry.org/quarks-statefulset/pkg/kube/client/clientset/versioned"
	"code.cloudfoundry.org/quarks-statefulset/pkg/kube/operator"
	"code.cloudfoundry.org/quarks-statefulset/testing"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	utils "code.cloudfoundry.org/quarks-utils/testing/integration"
	"code.cloudfoundry.org/quarks-utils/testing/machine"
)

// Environment starts our operator and handles interaction with the k8s
// cluster used in the tests
type Environment struct {
	*utils.Environment
	Machine
	testing.Catalog
}

var (
	namespaceCounter int32
)

const (
	defaultTestMeltdownDuration     = 10
	defaultTestMeltdownRequeueAfter = 1
)

// NewEnvironment returns a new struct
func NewEnvironment(kubeConfig *rest.Config) *Environment {
	atomic.AddInt32(&namespaceCounter, 1)
	namespaceID := gomegaConfig.GinkgoConfig.ParallelNode*200 + int(namespaceCounter)
	// the single namespace used by this test
	ns := utils.GetNamespaceName(namespaceID)

	env := &Environment{
		Environment: &utils.Environment{
			ID:         namespaceID,
			Namespace:  ns,
			KubeConfig: kubeConfig,
			Config: &config.Config{
				CtxTimeOut:           10 * time.Second,
				MeltdownDuration:     defaultTestMeltdownDuration * time.Second,
				MeltdownRequeueAfter: defaultTestMeltdownRequeueAfter * time.Second,
				MonitoredID:          ns,
				OperatorNamespace:    ns,
				Fs:                   afero.NewOsFs(),
			},
		},
		Machine: Machine{
			Machine: machine.NewMachine(),
		},
	}
	gomega.SetDefaultEventuallyTimeout(env.PollTimeout)
	gomega.SetDefaultEventuallyPollingInterval(env.PollInterval)

	return env
}

// SetupClientsets initializes kube clientsets
func (e *Environment) SetupClientsets() error {
	var err error
	e.Clientset, err = kubernetes.NewForConfig(e.KubeConfig)
	if err != nil {
		return err
	}

	e.VersionedClientset, err = versioned.NewForConfig(e.KubeConfig)
	if err != nil {
		return err
	}

	return nil
}

// ApplyCRDs applies the CRDs to the cluster
func ApplyCRDs(kubeConfig *rest.Config) error {
	err := operator.ApplyCRDs(context.Background(), kubeConfig)
	if err != nil {
		return err
	}
	return nil
}
