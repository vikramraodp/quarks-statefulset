package quarksstatefulset

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
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

		err := m.setNewOrdinal(ctx, updatedPod, podLabels)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
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

	azIndex, err := strconv.Atoi(podLabels[qstsv1a1.LabelAZIndex])
	if err != nil {
		azIndex = 0
	}
	specIndex := names.SpecIndex(azIndex, podOrdinal)

	if podOrdinal != -1 {
		podLabels[qstsv1a1.LabelPodOrdinal] = strconv.Itoa(podOrdinal)
		podLabels[qstsv1a1.LabelInstance] = strconv.Itoa(specIndex)
		pod.SetLabels(podLabels)
	}
}

func (m *PodMutator) setNewOrdinal(ctx context.Context, pod *corev1.Pod, podLabels map[string]string) error {
	labels := map[string]string{"controller-revision-hash": pod.Labels["controller-revision-hash"]}
	list := &corev1.PodList{}
	err := m.client.List(ctx, list, client.InNamespace(pod.Namespace), client.MatchingLabels(labels))
	if err != nil {
		return errors.Wrapf(err, "failed to list pods in namespace: %s", pod.Namespace)
	}

	newOrdinal := 0
	for _, p := range list.Items {
		if p.Name != pod.Name {
			newOrdinal++
		}
	}

	// zero based
	podLabels[qstsv1a1.LabelStartupOrdinal] = strconv.Itoa(newOrdinal)
	pod.SetLabels(podLabels)
	return nil
}

// isQuarksStatefulSet check is it is quarksStatefulSet Pod
func isQuarksStatefulSet(labels map[string]string) bool {
	if _, exists := labels[appsv1.StatefulSetPodNameLabel]; exists {
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
