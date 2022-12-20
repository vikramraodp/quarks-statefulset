package testing

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	qstsv1a1 "code.cloudfoundry.org/quarks-statefulset/pkg/kube/apis/quarksstatefulset/v1alpha1"
)

// DefaultQuarksStatefulSet for use in tests
func (c *Catalog) DefaultQuarksStatefulSet(name string) qstsv1a1.QuarksStatefulSet {
	return qstsv1a1.QuarksStatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qstsv1a1.QuarksStatefulSetSpec{
			Template: c.DefaultStatefulSet(name),
		},
	}
}

// QstsWithProbeSinglePod for use in tests
func (c *Catalog) QstsWithProbeSinglePod(name string, cmd []string) qstsv1a1.QuarksStatefulSet {
	return qstsv1a1.QuarksStatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qstsv1a1.QuarksStatefulSetSpec{
			Template: c.DefaultStatefulSet(name),
			ActivePassiveProbes: map[string]v1.Probe{
				"busybox": v1.Probe{
					PeriodSeconds: 2,
					ProbeHandler: v1.ProbeHandler{
						Exec: &v1.ExecAction{
							Command: cmd,
						},
					},
				}},
		},
	}
}

// QstsWithProbeSinglePodMultipleAZs for use in tests
func (c *Catalog) QstsWithProbeSinglePodMultipleAZs(name string, cmd []string, zones []string) qstsv1a1.QuarksStatefulSet {
	return qstsv1a1.QuarksStatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qstsv1a1.QuarksStatefulSetSpec{
			Template: c.DefaultStatefulSet(name),
			ActivePassiveProbes: map[string]v1.Probe{
				"busybox": v1.Probe{
					PeriodSeconds: 2,
					ProbeHandler: v1.ProbeHandler{
						Exec: &v1.ExecAction{
							Command: cmd,
						},
					},
				},
			},
			Zones: zones,
		},
	}
}

// QstsWithActiveSinglePod for use in tests
func (c *Catalog) QstsWithActiveSinglePod(name string, cmd []string) qstsv1a1.QuarksStatefulSet {
	return qstsv1a1.QuarksStatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qstsv1a1.QuarksStatefulSetSpec{
			Template: c.DefaultStatefulSetWithActiveSinglePod(name),
			ActivePassiveProbes: map[string]v1.Probe{
				"busybox": v1.Probe{
					PeriodSeconds: 2,
					ProbeHandler: v1.ProbeHandler{
						Exec: &v1.ExecAction{
							Command: cmd,
						},
					},
				}},
		},
	}
}

// QstsWithoutProbeMultiplePods for use in tests
func (c *Catalog) QstsWithoutProbeMultiplePods(name string, cmd []string) qstsv1a1.QuarksStatefulSet {
	return qstsv1a1.QuarksStatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qstsv1a1.QuarksStatefulSetSpec{
			Template: c.DefaultStatefulSetWithReplicasN(name),
			ActivePassiveProbes: map[string]v1.Probe{
				"busybox": v1.Probe{
					ProbeHandler: v1.ProbeHandler{
						Exec: &v1.ExecAction{
							Command: cmd,
						},
					},
				}},
		},
	}
}

// QstsWithProbeMultiplePods for use in tests
func (c *Catalog) QstsWithProbeMultiplePods(name string, cmd []string) qstsv1a1.QuarksStatefulSet {
	return qstsv1a1.QuarksStatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qstsv1a1.QuarksStatefulSetSpec{
			Template: c.DefaultStatefulSetWithReplicasN(name),
			ActivePassiveProbes: map[string]v1.Probe{
				"busybox": v1.Probe{
					PeriodSeconds: 2,
					ProbeHandler: v1.ProbeHandler{
						Exec: &v1.ExecAction{
							Command: cmd,
						},
					},
				}},
		},
	}
}

// WrongQuarksStatefulSet for use in tests
func (c *Catalog) WrongQuarksStatefulSet(name string) qstsv1a1.QuarksStatefulSet {
	return qstsv1a1.QuarksStatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qstsv1a1.QuarksStatefulSetSpec{
			Template: c.WrongStatefulSet(name),
		},
	}
}

// QuarksStatefulSetWithPVC for use in tests
func (c *Catalog) QuarksStatefulSetWithPVC(name, pvcName string, storageClassName string) qstsv1a1.QuarksStatefulSet {
	return qstsv1a1.QuarksStatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qstsv1a1.QuarksStatefulSetSpec{
			Template: c.StatefulSetWithPVC(name, pvcName, storageClassName),
		},
	}
}

// WrongQuarksStatefulSetWithPVC for use in tests
func (c *Catalog) WrongQuarksStatefulSetWithPVC(name, pvcName string, storageClassName string) qstsv1a1.QuarksStatefulSet {
	return qstsv1a1.QuarksStatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qstsv1a1.QuarksStatefulSetSpec{
			Template: c.WrongStatefulSetWithPVC(name, pvcName, storageClassName),
		},
	}
}

// OwnedReferencesQuarksStatefulSet for use in tests
func (c *Catalog) OwnedReferencesQuarksStatefulSet(name string) qstsv1a1.QuarksStatefulSet {
	return qstsv1a1.QuarksStatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: qstsv1a1.QuarksStatefulSetSpec{
			UpdateOnConfigChange: true,
			Template:             c.OwnedReferencesStatefulSet(name),
		},
	}
}
