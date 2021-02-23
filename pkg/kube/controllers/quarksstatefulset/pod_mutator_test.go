package quarksstatefulset_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"gomodules.xyz/jsonpatch/v2"

	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	qstsv1a1 "code.cloudfoundry.org/quarks-statefulset/pkg/kube/apis/quarksstatefulset/v1alpha1"
	"code.cloudfoundry.org/quarks-statefulset/pkg/kube/controllers/quarksstatefulset"
	"code.cloudfoundry.org/quarks-statefulset/testing"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	helper "code.cloudfoundry.org/quarks-utils/testing/testhelper"
)

var _ = Describe("Add labels to qsts pods", func() {
	var (
		client   client.Client
		ctx      context.Context
		decoder  *admission.Decoder
		env      testing.Catalog
		log      *zap.SugaredLogger
		mutator  admission.Handler
		pod      corev1.Pod
		request  admission.Request
		response admission.Response
		qsts     qstsv1a1.QuarksStatefulSet
		scheme   *runtime.Scheme
	)

	revisionPod := func(name string, revision string) corev1.Pod {
		return env.LabeledPod(name, map[string]string{
			appsv1.StatefulSetPodNameLabel: "exists",
			qstsv1a1.LabelQStsName:         "mutate-test",
			"controller-revision-hash":     revision,
		})
	}

	revisionSts := func(revision string) *appsv1.StatefulSet {
		return &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mutate-test-0",
				Labels: map[string]string{
					qstsv1a1.LabelQStsName: "mutate-test",
				},
			},
			Status: appsv1.StatefulSetStatus{
				CurrentRevision: revision,
			},
		}
	}

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
			AdmissionRequest: admissionv1.AdmissionRequest{
				Object: runtime.RawExtension{Raw: raw},
			},
		}
	}

	BeforeEach(func() {
		_, log = helper.NewTestLogger()
		ctx = ctxlog.NewParentContext(log)

		mutator = quarksstatefulset.NewPodMutator(log, &config.Config{CtxTimeOut: 10 * time.Second})

		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(appsv1.AddToScheme(scheme)).To(Succeed())
		Expect(qstsv1a1.AddToScheme(scheme)).To(Succeed())

		decoder, _ = admission.NewDecoder(scheme)
		_ = mutator.(admission.DecoderInjector).InjectDecoder(decoder)

		qsts = env.DefaultQuarksStatefulSet("mutate-test")
	})

	JustBeforeEach(func() {
		_ = mutator.(inject.Client).InjectClient(client)
		response = mutator.Handle(ctx, request)
	})

	When("pod is not part of qsts", func() {
		BeforeEach(func() {
			pod = corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "name"},
				Spec:       env.Sleep1hPodSpec(),
			}
			request = newAdmissionRequest(pod)
			client = fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithObjects(&pod).
				Build()
		})

		It("does not modify", func() {
			Expect(response.AdmissionResponse.Allowed).To(BeTrue())
			Expect(response.Patches).To(BeEmpty())
		})
	})

	When("pod is part of qsts", func() {
		When("list pods would return none, pod is not created yet", func() {
			BeforeEach(func() {
				pod = revisionPod("qsts-pod-0", "abcd")
				client = fake.
					NewClientBuilder().
					WithScheme(scheme).
					WithObjects(&qsts).
					Build()
				request = newAdmissionRequest(pod)
			})

			It("sets updates revisions annotation on qsts", func() {
				qsts := &qstsv1a1.QuarksStatefulSet{}
				err := client.Get(context.TODO(), types.NamespacedName{Name: "mutate-test", Namespace: ""}, qsts)
				Expect(err).ToNot(HaveOccurred())
				Expect(qsts.Annotations[qstsv1a1.AnnotationRevisions]).To(Equal(`{"abcd":{"0":"0"}}`))
			})

			It("sets ordinal labels correct", func() {
				Expect(response.Allowed).To(BeTrue(), fmt.Sprintf("%v", response.Result))

				Expect(response.Patches).To(HaveLen(2))
				patches := jsonPatches(response.Patches)
				Expect(patches).To(ContainElement(addLabelPatch("pod-ordinal", "0")))
				Expect(patches).To(ContainElement(addLabelPatch("startup-ordinal", "0")))

				Expect(response.AdmissionResponse.Allowed).To(BeTrue())
			})
		})

		When("it is the first pod", func() {
			BeforeEach(func() {
				pod = revisionPod("qsts-pod-0", "abcd")
				client = fake.
					NewClientBuilder().
					WithScheme(scheme).
					WithObjects(&qsts, &pod).
					Build()
				request = newAdmissionRequest(pod)
			})

			It("sets ordinal labels correct", func() {
				Expect(response.Allowed).To(BeTrue(), fmt.Sprintf("%v", response.Result))

				Expect(response.Patches).To(HaveLen(2))
				patches := jsonPatches(response.Patches)
				Expect(patches).To(ContainElement(addLabelPatch("pod-ordinal", "0")))
				Expect(patches).To(ContainElement(addLabelPatch("startup-ordinal", "0")))

				Expect(response.AdmissionResponse.Allowed).To(BeTrue())
			})
		})

		When("another pod is added", func() {
			BeforeEach(func() {
				sts := revisionSts("abcd")
				pod = revisionPod("qsts-pod-0", "abcd")
				client = fake.
					NewClientBuilder().
					WithScheme(scheme).
					WithObjects(&qsts, sts, &pod).
					Build()
				request = newAdmissionRequest(pod)
			})

			It("keeps revision annotation updated", func() {
				Expect(response.Allowed).To(BeTrue(), fmt.Sprintf("%v", response.Result))

				qsts := &qstsv1a1.QuarksStatefulSet{}
				err := client.Get(context.TODO(), types.NamespacedName{Name: "mutate-test", Namespace: ""}, qsts)
				Expect(err).ToNot(HaveOccurred())
				Expect(qsts.Annotations[qstsv1a1.AnnotationRevisions]).To(Equal(`{"abcd":{"0":"0"}}`))

				first := revisionPod("qsts-pod-1", "abcd")
				err = client.Create(context.TODO(), &first)
				Expect(err).ToNot(HaveOccurred())

				request = newAdmissionRequest(first)
				response = mutator.Handle(ctx, request)
				Expect(response.Allowed).To(BeTrue(), fmt.Sprintf("%v", response.Result))

				err = client.Get(context.TODO(), types.NamespacedName{Name: "mutate-test", Namespace: ""}, qsts)
				Expect(err).ToNot(HaveOccurred())
				Expect(qsts.Annotations[qstsv1a1.AnnotationRevisions]).To(Equal(`{"abcd":{"0":"0","1":"1"}}`))
			})
		})

		When("pod-0 exists first", func() {
			BeforeEach(func() {
				pod = revisionPod("qsts-pod-1", "abcd")
				first := revisionPod("qsts-pod-0", "abcd")
				client = fake.
					NewClientBuilder().
					WithScheme(scheme).
					WithObjects(&qsts, &pod, &first).
					Build()
				request = newAdmissionRequest(pod)
			})

			It("sets ordinal labels are correct", func() {
				Expect(response.Allowed).To(BeTrue(), fmt.Sprintf("%v", response.Result))

				Expect(response.Patches).To(HaveLen(2))
				patches := jsonPatches(response.Patches)
				Expect(patches).To(ContainElement(addLabelPatch("pod-ordinal", "1")))
				Expect(patches).To(ContainElement(addLabelPatch("startup-ordinal", "1")))

				Expect(response.AdmissionResponse.Allowed).To(BeTrue())
			})
		})

		When("pod-1 exists first", func() {
			When("this is a new deployment", func() {
				BeforeEach(func() {
					pod = revisionPod("qsts-pod-0", "abcd")
					first := revisionPod("qsts-pod-1", "abcd")
					client = fake.
						NewClientBuilder().
						WithScheme(scheme).
						WithObjects(&qsts, &pod, &first).
						Build()
					request = newAdmissionRequest(pod)
				})

				It("sets ordinal labels correctly", func() {
					Expect(response.Allowed).To(BeTrue(), fmt.Sprintf("%v", response.Result))

					Expect(response.Patches).To(HaveLen(2))
					patches := jsonPatches(response.Patches)
					Expect(patches).To(ContainElement(addLabelPatch("pod-ordinal", "0")))
					// first created pod is bootstrapping,
					// this pod was created second, so
					// startup-ordinal should be 1
					Expect(patches).To(ContainElement(addLabelPatch("startup-ordinal", "1")))

					Expect(response.AdmissionResponse.Allowed).To(BeTrue())
				})
			})

			When("restarting the first pod", func() {
				BeforeEach(func() {
					pod = revisionPod("qsts-pod-0", "abcd")
					first := revisionPod("qsts-pod-1", "abcd")

					_ = qsts.SetRevisions(qstsv1a1.Revisions{"abcd": qstsv1a1.Ordinals{"0": "0", "1": "1"}})
					sts := revisionSts("abcd")

					client = fake.
						NewClientBuilder().
						WithScheme(scheme).
						WithObjects(&qsts, sts, &pod, &first).
						Build()
					request = newAdmissionRequest(pod)
				})

				It("keeps the previous startup-ordinal", func() {
					Expect(response.Allowed).To(BeTrue(), fmt.Sprintf("%v", response.Result))

					Expect(response.Patches).To(HaveLen(2))
					patches := jsonPatches(response.Patches)
					Expect(patches).To(ContainElement(addLabelPatch("pod-ordinal", "0")))
					// this is just a restart
					Expect(patches).To(ContainElement(addLabelPatch("startup-ordinal", "0")))

					Expect(response.AdmissionResponse.Allowed).To(BeTrue())
				})
			})
		})

		When("pod from different controller revision hash exists", func() {
			BeforeEach(func() {
				pod = revisionPod("qsts-pod-1", "efgh")
				first := revisionPod("qsts-pod-0", "abcd")
				client = fake.
					NewClientBuilder().
					WithScheme(scheme).
					WithObjects(&qsts, &pod, &first).
					Build()
				request = newAdmissionRequest(pod)
			})

			It("sets ordinal labels are correct", func() {
				Expect(response.Allowed).To(BeTrue(), fmt.Sprintf("%v", response.Result))

				Expect(response.Patches).To(HaveLen(2))
				patches := jsonPatches(response.Patches)
				Expect(patches).To(ContainElement(addLabelPatch("pod-ordinal", "1")))
				Expect(patches).To(ContainElement(addLabelPatch("startup-ordinal", "0")))

				Expect(response.AdmissionResponse.Allowed).To(BeTrue())
			})
		})

		When("old controller revision vanished", func() {
			BeforeEach(func() {
				pod = revisionPod("qsts-pod-1", "efgh")
				_ = qsts.SetRevisions(qstsv1a1.Revisions{"abcd": qstsv1a1.Ordinals{"0": "0", "1": "1"}})
				client = fake.
					NewClientBuilder().
					WithScheme(scheme).
					WithObjects(&qsts, &pod).
					Build()
				request = newAdmissionRequest(pod)
			})

			It("adds the new revision to the annotation on qsts and cleans up", func() {
				qsts := &qstsv1a1.QuarksStatefulSet{}
				err := client.Get(context.TODO(), types.NamespacedName{Name: "mutate-test", Namespace: ""}, qsts)
				Expect(err).ToNot(HaveOccurred())
				Expect(qsts.Annotations[qstsv1a1.AnnotationRevisions]).To(Equal(`{"efgh":{"1":"0"}}`))
			})
		})

		When("old controller revision exists", func() {
			BeforeEach(func() {
				pod = revisionPod("qsts-pod-1", "efgh")
				first := revisionPod("qsts-pod-0", "abcd")
				_ = qsts.SetRevisions(qstsv1a1.Revisions{"abcd": qstsv1a1.Ordinals{"0": "0", "1": "1"}})
				sts := revisionSts("abcd")
				client = fake.
					NewClientBuilder().
					WithScheme(scheme).
					WithObjects(&qsts, sts, &pod, &first).
					Build()
				request = newAdmissionRequest(pod)
			})

			It("adds the new revision to the annotation on qsts", func() {
				qsts := &qstsv1a1.QuarksStatefulSet{}
				err := client.Get(context.TODO(), types.NamespacedName{Name: "mutate-test", Namespace: ""}, qsts)
				Expect(err).ToNot(HaveOccurred())
				Expect(qsts.Annotations[qstsv1a1.AnnotationRevisions]).To(Equal(`{"abcd":{"0":"0","1":"1"},"efgh":{"1":"0"}}`))
			})
		})
	})
})
