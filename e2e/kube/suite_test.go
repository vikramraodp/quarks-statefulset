package kube_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cmdHelper "code.cloudfoundry.org/quarks-utils/testing"
	"code.cloudfoundry.org/quarks-utils/testing/e2ehelper"
)

const examplesDir = "../../docs/examples/"

var (
	nsIndex           int
	teardowns         []e2ehelper.TearDownFunc
	namespace         string
	operatorNamespace string
	kubectl           *cmdHelper.Kubectl
)

func FailAndCollectDebugInfo(description string, callerSkip ...int) {
	Fail(description, callerSkip...)
}

func TestE2EKube(t *testing.T) {
	nsIndex = 0
	kubectl = cmdHelper.NewKubectl()
	SetDefaultEventuallyTimeout(kubectl.PollTimeout)
	SetDefaultEventuallyPollingInterval(500 * time.Millisecond)

	RegisterFailHandler(FailAndCollectDebugInfo)
	RunSpecs(t, "E2E Kube Suite")
}

var _ = BeforeEach(func() {
	var err error
	var teardown e2ehelper.TearDownFunc

	dir, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())

	chartPath := fmt.Sprintf("%s%s", dir, "/../../helm/quarks-statefulset")

	namespace, operatorNamespace, teardown, err = e2ehelper.CreateNamespace()
	Expect(err).ToNot(HaveOccurred())
	teardowns = append(teardowns, teardown)

	teardown, err = e2ehelper.CreateMonitoredNamespace(namespace, operatorNamespace)
	Expect(err).ToNot(HaveOccurred())
	teardowns = append(teardowns, teardown)

	teardown, err = e2ehelper.InstallChart(chartPath, operatorNamespace,
		"--set", fmt.Sprintf("global.monitoredID=%s", operatorNamespace),
	)
	Expect(err).ToNot(HaveOccurred())
	// prepend helm clean up
	teardowns = append([]e2ehelper.TearDownFunc{teardown}, teardowns...)
})

var _ = AfterEach(func() {
	err := e2ehelper.TearDownAll(teardowns)
	if err != nil {
		fmt.Printf("Failures while cleaning up test environment:\n %v", err)
	}
	teardowns = []e2ehelper.TearDownFunc{}
})

var _ = AfterSuite(func() {
	err := e2ehelper.TearDownAll(teardowns)
	if err != nil {
		fmt.Printf("Failures while cleaning up test environment:\n %v", err)
	}
})

func waitReady(name string) {
	waitReadyNamespace(namespace, name)
}

func waitReadyNamespace(ns, name string) {
	err := kubectl.Wait(ns, "ready", name, kubectl.PollTimeout)
	Expect(err).ToNot(HaveOccurred(), "waiting for resource: ", name)
}
