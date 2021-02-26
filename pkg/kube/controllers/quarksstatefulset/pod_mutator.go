package quarksstatefulset

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	qstsv1a1 "code.cloudfoundry.org/quarks-statefulset/pkg/kube/apis/quarksstatefulset/v1alpha1"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/names"
)

// PodMutator for adding the pod-ordinal label on statefulset pods
type PodMutator struct {
	client  client.Client
	log     *zap.SugaredLogger
	config  *config.Config
	decoder *admission.Decoder
}

// Check that PodMutator implements the admission.Handler interface
var _ admission.Handler = &PodMutator{}

// NewPodMutator returns a pod mutator to add pod-ordinal on statefulset pods
func NewPodMutator(log *zap.SugaredLogger, config *config.Config) admission.Handler {
	mutatorLog := log.Named("quarks-statefulset-pod-mutator")
	mutatorLog.Info("Creating a Pod mutator for QuarksStatefulSet")

	return &PodMutator{
		log:    mutatorLog,
		config: config,
	}
}

// Handle checks if pod is part of a statefulset and adds the pod-ordinal labels
// on the pod for service selectors
func (m *PodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	err := m.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	updatedPod := pod.DeepCopy()
	if isQuarksStatefulSet(pod.GetLabels()) {
		m.log.Debugf("Mutating pod '%s/%s' (%s), adding ordinals", pod.Namespace, pod.Name, req.Namespace)
		podLabels := pod.GetLabels()
		if podLabels == nil {
			podLabels = map[string]string{}
		}
		setPodOrdinal(updatedPod, podLabels)
	}

	marshaledPod, err := json.Marshal(updatedPod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// setPodOrdinal adds a pod-ordinal label to the given pod
func setPodOrdinal(pod *corev1.Pod, podLabels map[string]string) {
	podOrdinal := names.OrdinalFromPodName(pod.GetName())

	if podOrdinal != -1 {
		podLabels[qstsv1a1.LabelPodOrdinal] = strconv.Itoa(podOrdinal)
		pod.SetLabels(podLabels)
	}
}

// isQuarksStatefulSet check is it is quarksStatefulSet Pod
func isQuarksStatefulSet(labels map[string]string) bool {
	_, sts := labels[appsv1.StatefulSetPodNameLabel]
	_, qsts := labels[qstsv1a1.LabelQStsName]
	if sts && qsts {
		return true
	}
	return false
}

// Check that PodMutator implements the admission.DecoderInjector interface
var _ admission.DecoderInjector = &PodMutator{}

// InjectDecoder injects the decoder.
func (m *PodMutator) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d
	return nil
}

// Check that PodMutator implements the inject.Client interface
var _ inject.Client = &PodMutator{}

// InjectClient injects the client.
func (m *PodMutator) InjectClient(c client.Client) error {
	m.client = c
	return nil
}
