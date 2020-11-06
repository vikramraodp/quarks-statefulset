package quarksstatefulset_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"gomodules.xyz/jsonpatch/v2"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"code.cloudfoundry.org/quarks-statefulset/pkg/kube/controllers/quarksstatefulset"
	"code.cloudfoundry.org/quarks-statefulset/testing"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	helper "code.cloudfoundry.org/quarks-utils/testing/testhelper"
)

var _ = Describe("Add labels to qsts pods", func() {
	var (
		ctx      context.Context
		decoder  *admission.Decoder
		env      testing.Catalog
		log      *zap.SugaredLogger
		mutator  admission.Handler
		pod      corev1.Pod
		request  admission.Request
		response admission.Response
	)

	addLabelPatch := func(name string, value string) string {
		return fmt.Sprintf(`{"op":"add","path":"/metadata/labels/quarks.cloudfoundry.org~1%s","value":"%s"}`, name, value)
	}

	jsonPatches := func(operations []jsonpatch.Operation) []string {
		patches := make([]string, len(operations))
		for i, patch := range operations {
			patches[i] = patch.Json()
		}
		return patches
	}

	newAdmissionRequest := func(pod corev1.Pod) admission.Request {
		raw, _ := json.Marshal(pod)
		return admission.Request{
			AdmissionRequest: admissionv1beta1.AdmissionRequest{
				Object: runtime.RawExtension{Raw: raw},
			},
		}
	}

	BeforeEach(func() {
		_, log = helper.NewTestLogger()
		ctx = ctxlog.NewParentContext(log)

		mutator = quarksstatefulset.NewPodMutator(log, &config.Config{CtxTimeOut: 10 * time.Second})

		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())

		decoder, _ = admission.NewDecoder(scheme)
		_ = mutator.(admission.DecoderInjector).InjectDecoder(decoder)
	})

	JustBeforeEach(func() {
		response = mutator.Handle(ctx, request)
	})

	When("pod is not part of qsts", func() {
		BeforeEach(func() {
			pod = corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "name",
				},
				Spec: env.Sleep1hPodSpec(),
			}
			request = newAdmissionRequest(pod)
		})

		It("does not modify", func() {
			Expect(response.AdmissionResponse.Allowed).To(BeTrue())
			Expect(response.Patches).To(BeEmpty())
		})
	})

	When("pod is part of qsts", func() {
		When("it is not created yet", func() {
			BeforeEach(func() {
				pod = env.LabeledPod("qsts-pod-0", map[string]string{
					appsv1.StatefulSetPodNameLabel: "exists",
					"controller-revision-hash":     "abcd",
				})
				request = newAdmissionRequest(pod)
			})

			It("sets ordinal labels correct", func() {
				Expect(response.Allowed).To(BeTrue(), fmt.Sprintf("%v", response.Result))

				Expect(response.Patches).To(HaveLen(2))
				patches := jsonPatches(response.Patches)
				Expect(patches).To(ContainElement(addLabelPatch("pod-ordinal", "0")))
				Expect(patches).To(ContainElement(addLabelPatch("spec-index", "0")))

				Expect(response.AdmissionResponse.Allowed).To(BeTrue())
			})
		})

		When("it is the first pod", func() {
			BeforeEach(func() {
				pod = env.LabeledPod("qsts-pod-0", map[string]string{
					appsv1.StatefulSetPodNameLabel: "exists",
					"controller-revision-hash":     "abcd",
				})
				request = newAdmissionRequest(pod)
			})

			It("sets ordinal labels correct", func() {
				Expect(response.Allowed).To(BeTrue(), fmt.Sprintf("%v", response.Result))

				Expect(response.Patches).To(HaveLen(2))
				patches := jsonPatches(response.Patches)
				Expect(patches).To(ContainElement(addLabelPatch("pod-ordinal", "0")))
				Expect(patches).To(ContainElement(addLabelPatch("spec-index", "0")))

				Expect(response.AdmissionResponse.Allowed).To(BeTrue())
			})
		})

		When("pod-0 exists first", func() {
			BeforeEach(func() {
				pod = env.LabeledPod("qsts-pod-1", map[string]string{
					appsv1.StatefulSetPodNameLabel: "exists",
					"controller-revision-hash":     "abcd",
				})
				request = newAdmissionRequest(pod)
			})

			It("sets ordinal labels are correct", func() {
				Expect(response.Allowed).To(BeTrue(), fmt.Sprintf("%v", response.Result))

				Expect(response.Patches).To(HaveLen(2))
				patches := jsonPatches(response.Patches)
				Expect(patches).To(ContainElement(addLabelPatch("pod-ordinal", "1")))
				Expect(patches).To(ContainElement(addLabelPatch("spec-index", "1")))

				Expect(response.AdmissionResponse.Allowed).To(BeTrue())
			})
		})
	})
})
