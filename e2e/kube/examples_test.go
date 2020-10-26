package kube_test

import (
	"path"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/wait"

	cmdHelper "code.cloudfoundry.org/quarks-utils/testing"
)

var _ = Describe("Examples Directory", func() {
	var (
		example      string
		yamlFilePath string
	)

	const pollInterval = 5 * time.Second

	JustBeforeEach(func() {
		yamlFilePath = example
		if !strings.HasPrefix(example, "/") {
			yamlFilePath = path.Join(examplesDir, example)
		}
		err := cmdHelper.Create(namespace, yamlFilePath)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("quarks-statefulset configs examples", func() {
		BeforeEach(func() {
			example = "qstatefulset_configs.yaml"
		})

		It("creates and updates statefulsets", func() {
			By("Checking for pods")
			waitReady("pod/example-quarks-statefulset-0")
			waitReady("pod/example-quarks-statefulset-1")

			yamlUpdatedFilePath := examplesDir + "qstatefulset_configs_updated.yaml"

			By("Updating the config value used by pods")
			err := cmdHelper.Apply(namespace, yamlUpdatedFilePath)
			Expect(err).ToNot(HaveOccurred())

			By("Checking the updated value in the env")
			err = wait.PollImmediate(pollInterval, kubectl.PollTimeout, func() (bool, error) {
				err := kubectl.RunCommandWithCheckString(namespace, "example-quarks-statefulset-0", "env", "SPECIAL_KEY=value1Updated")
				if err != nil {
					return false, nil
				}
				return true, nil
			})
			Expect(err).ToNot(HaveOccurred(), "polling for example-quarks-statefulset-0 with special key")

			err = kubectl.RunCommandWithCheckString(namespace, "example-quarks-statefulset-1", "env", "SPECIAL_KEY=value1Updated")
			Expect(err).ToNot(HaveOccurred(), "waiting for example-quarks-statefulset-1 with special key")
		})

		It("creates and updates statefulsets even out of a failure situation", func() {
			By("Checking for pods")
			waitReady("pod/example-quarks-statefulset-0")
			waitReady("pod/example-quarks-statefulset-1")

			yamlUpdatedFilePathFailing := examplesDir + "qstatefulset_configs_fail.yaml"
			err := cmdHelper.Apply(namespace, yamlUpdatedFilePathFailing)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for failed pod")
			err = wait.PollImmediate(pollInterval, kubectl.PollTimeout, func() (bool, error) {
				podStatus, err := kubectl.PodStatus(namespace, "example-quarks-statefulset-1")
				if err != nil {
					return true, err
				}

				return len(podStatus.ContainerStatuses) > 0 &&
					podStatus.ContainerStatuses[0].LastTerminationState.Terminated != nil &&
					podStatus.ContainerStatuses[0].LastTerminationState.Terminated.ExitCode == 1, nil
			})
			Expect(err).ToNot(HaveOccurred(), "polling for example-quarks-statefulset-1")

			yamlUpdatedFilePath := examplesDir + "qstatefulset_configs_updated.yaml"

			By("Updating the config value used by pods")
			err = cmdHelper.Apply(namespace, yamlUpdatedFilePath)
			Expect(err).ToNot(HaveOccurred())

			By("Checking the updated value in the env")
			err = wait.PollImmediate(pollInterval, kubectl.PollTimeout, func() (bool, error) {
				err := kubectl.RunCommandWithCheckString(namespace, "example-quarks-statefulset-0", "env", "SPECIAL_KEY=value1Updated")
				if err != nil {
					return false, nil
				}
				return true, nil
			})
			Expect(err).ToNot(HaveOccurred(), "polling for example-quarks-statefulset-0 with special key")

			err = wait.PollImmediate(pollInterval, kubectl.PollTimeout, func() (bool, error) {
				err := kubectl.RunCommandWithCheckString(namespace, "example-quarks-statefulset-1", "env", "SPECIAL_KEY=value1Updated")
				if err != nil {
					return false, nil
				}
				return true, nil
			})
			Expect(err).ToNot(HaveOccurred(), "polling for example-quarks-statefulset-1 with special key")
		})

		It("it labels the first pod as active", func() {
			yamlUpdatedFilePath := examplesDir + "qstatefulset_active_passive.yaml"
			By("Applying a quarkstatefulset with active-passive probe")
			err := cmdHelper.Apply(namespace, yamlUpdatedFilePath)
			Expect(err).ToNot(HaveOccurred())

			err = kubectl.WaitForPod(namespace, "quarks.cloudfoundry.org/pod-active", "example-quarks-statefulset-0")
			Expect(err).ToNot(HaveOccurred(), "waiting for example-quarks-statefulset-0")
		})

	})

	Context("quarks-statefulset examples", func() {
		BeforeEach(func() {
			example = "qstatefulset_tolerations.yaml"
		})

		It("creates statefulset pods with tolerations defined", func() {
			By("Checking for pods")
			waitReady("pod/example-quarks-statefulset-0")

			tolerations, err := cmdHelper.GetData(namespace, "pod", "example-quarks-statefulset-0", "go-template={{.spec.tolerations}}")
			Expect(err).ToNot(HaveOccurred())
			Expect(tolerations).To(ContainSubstring(string("effect:NoSchedule")))
			Expect(tolerations).To(ContainSubstring(string("key:key")))
			Expect(tolerations).To(ContainSubstring(string("value:value")))

		})
	})
})
