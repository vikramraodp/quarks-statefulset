package environment

import (
	"code.cloudfoundry.org/quarks-utils/pkg/monitorednamespace"
	utils "code.cloudfoundry.org/quarks-utils/testing/integration"
)

// SetupNamespace creates the namespace and the clientsets and prepares the teardowm
func (e *Environment) SetupNamespace() error {
	return utils.SetupNamespace(e.Environment, e.Machine.Machine,
		map[string]string{
			monitorednamespace.LabelNamespace: e.Config.MonitoredID,
		})
}
