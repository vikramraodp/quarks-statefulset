package quarksstatefulset

import (
	"context"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	crc "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	qstsv1a1 "code.cloudfoundry.org/quarks-statefulset/pkg/kube/apis/quarksstatefulset/v1alpha1"
	"code.cloudfoundry.org/quarks-statefulset/pkg/kube/util/reference"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"code.cloudfoundry.org/quarks-utils/pkg/monitorednamespace"
	"code.cloudfoundry.org/quarks-utils/pkg/skip"
	vss "code.cloudfoundry.org/quarks-utils/pkg/versionedsecretstore"
)

// AddQuarksStatefulSet creates a new QuarksStatefulSet controller to watch for the custom resource and
// reconcile it into statefulSets.
func AddQuarksStatefulSet(ctx context.Context, config *config.Config, mgr manager.Manager) error {
	ctx = ctxlog.NewContextWithRecorder(ctx, "quarks-statefulset-reconciler", mgr.GetEventRecorderFor("quarks-statefulset-recorder"))
	store := vss.NewVersionedSecretStore(mgr.GetClient())
	r := NewReconciler(ctx, config, mgr, controllerutil.SetControllerReference, store)

	// Create a new controller
	c, err := controller.New("quarks-statefulset-controller", mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: config.MaxQuarksStatefulSetWorkers,
	})
	if err != nil {
		return errors.Wrap(err, "Adding QuarksStatefulSet controller to manager failed.")
	}

	client, err := appsv1client.NewForConfig(mgr.GetConfig())
	if err != nil {
		return errors.Wrap(err, "Could not get kube client")
	}

	nsPred := monitorednamespace.NewNSPredicate(ctx, mgr.GetClient(), config.MonitoredID)

	// Watch for changes to primary resource QuarksStatefulSet
	// Trigger when
	// - create event of quarksStatefulSet which have no children resources
	// - update event of quarksStatefulSet
	p := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			o := e.Object.(*qstsv1a1.QuarksStatefulSet)
			sts, err := listStatefulSetsFromAPIClient(ctx, client, o)
			if err != nil {
				ctxlog.Errorf(ctx, "Failed to list StatefulSets owned by QuarksStatefulSet '%s': %s", o.GetNamespacedName(), err)
			}
			if len(sts) == 0 {
				ctxlog.NewPredicateEvent(e.Object).Debug(
					ctx, e.Object, "qstsv1a1.QuarksStatefulSet",
					fmt.Sprintf("Create predicate passed for '%s/%s'", e.Object.GetNamespace(), e.Object.GetName()),
				)
				return true
			}

			return false
		},
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			o := e.ObjectOld.(*qstsv1a1.QuarksStatefulSet)
			n := e.ObjectNew.(*qstsv1a1.QuarksStatefulSet)

			if !reflect.DeepEqual(o.Spec.Template.Spec.VolumeClaimTemplates, n.Spec.Template.Spec.VolumeClaimTemplates) {
				ctxlog.WithEvent(n, "VolumeClaimTemplatesWarning").Infof(ctx, "Change in VolumeClaimTemplates QuarksStatefulSet won't be performed in sts as it's not supported by Kubernetes")
			}

			// don't trigger for update to Annotations
			if !reflect.DeepEqual(o.Spec, n.Spec) || !reflect.DeepEqual(o.Labels, n.Labels) {
				ctxlog.NewPredicateEvent(e.ObjectNew).Debug(
					ctx, e.ObjectNew, "qstsv1a1.QuarksStatefulSet",
					fmt.Sprintf("Update predicate passed for '%s/%s'", e.ObjectNew.GetNamespace(), e.ObjectNew.GetName()),
				)
				return true
			}
			return false
		},
	}
	err = c.Watch(&source.Kind{Type: &qstsv1a1.QuarksStatefulSet{}}, &handler.EnqueueRequestForObject{}, nsPred, p)
	if err != nil {
		return err
	}

	// Watch ConfigMaps referenced by the QuarksStatefulSet
	configMapPredicates := predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return true },
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldConfigMap := e.ObjectOld.(*corev1.ConfigMap)
			newConfigMap := e.ObjectNew.(*corev1.ConfigMap)

			return !reflect.DeepEqual(oldConfigMap.Data, newConfigMap.Data)
		},
	}
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, handler.EnqueueRequestsFromMapFunc(
		func(a crc.Object) []reconcile.Request {
			config := a.(*corev1.ConfigMap)

			if skip.Reconciles(ctx, mgr.GetClient(), config) {
				return []reconcile.Request{}
			}

			reconciles, err := reference.GetReconciles(ctx, mgr.GetClient(), config, false)
			if err != nil {
				ctxlog.Errorf(ctx, "Failed to calculate reconciles for configMap '%s/%s': %v", config.Namespace, config.Name, err)
			}

			for _, reconciliation := range reconciles {
				ctxlog.NewMappingEvent(a).Debug(ctx, reconciliation, "QuarksStatefulSet", a.GetName(), "config-maps")
			}
			return reconciles
		}),
		nsPred, configMapPredicates)
	if err != nil {
		return errors.Wrapf(err, "Watching configMaps failed in QuarksStatefulSet controller failed.")
	}

	// Watch Secrets referenced by the QuarksStatefulSet
	secretPredicates := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {

			o := e.Object.(*corev1.Secret)
			if !vss.IsVersionedSecret(*o) {
				return false
			}

			reconciles, err := reference.GetReconciles(ctx, mgr.GetClient(), o, true)
			if err != nil {
				ctxlog.Errorf(ctx, "Failed to calculate reconciles for secret '%s/%s': %v", o.Namespace, o.Name, err)
			}

			return len(reconciles) > 0
		},
		DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		GenericFunc: func(e event.GenericEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldSecret := e.ObjectOld.(*corev1.Secret)
			newSecret := e.ObjectNew.(*corev1.Secret)

			return !reflect.DeepEqual(oldSecret.Data, newSecret.Data)
		},
	}
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, handler.EnqueueRequestsFromMapFunc(
		func(a crc.Object) []reconcile.Request {
			secret := a.(*corev1.Secret)

			if skip.Reconciles(ctx, mgr.GetClient(), secret) {
				return []reconcile.Request{}
			}

			reconciles, err := reference.GetReconciles(ctx, mgr.GetClient(), secret, false)
			if err != nil {
				ctxlog.Errorf(ctx, "Failed to calculate reconciles for secret '%s/%s': %v", secret.Namespace, secret.Name, err)
			}

			for _, reconciliation := range reconciles {
				ctxlog.NewMappingEvent(a).Debug(ctx, reconciliation, "QuarksStatefulSet", a.GetName(), "secret")
			}

			return reconciles
		}), nsPred, secretPredicates)
	if err != nil {
		return errors.Wrapf(err, "Watching secrets failed in QuarksStatefulSet controller failed.")
	}

	return nil
}
