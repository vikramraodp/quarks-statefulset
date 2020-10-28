package controllers

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	qstsv1a1 "code.cloudfoundry.org/quarks-statefulset/pkg/kube/apis/quarksstatefulset/v1alpha1"
	"code.cloudfoundry.org/quarks-statefulset/pkg/kube/controllers/quarksstatefulset"
	"code.cloudfoundry.org/quarks-statefulset/pkg/kube/controllers/statefulset"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/credsgen"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"code.cloudfoundry.org/quarks-utils/pkg/webhook"
)

const (
	// HTTPReadyzEndpoint route
	HTTPReadyzEndpoint = "/readyz"
	// WebhookConfigPrefix is the prefix for the dir containing the webhook SSL certs
	WebhookConfigPrefix = "qsts-hook-"
)

// Theses funcs construct controllers and add them to the controller-runtime
// manager. The manager will set fields on the controllers and start them, when
// itself is started.
var addToManagerFuncs = []func(context.Context, *config.Config, manager.Manager) error{
	quarksstatefulset.AddQuarksStatefulSet,
	quarksstatefulset.AddQuarksStatefulSetStatus,
	statefulset.AddStatefulSetRollout,
	quarksstatefulset.AddStatefulSetActivePassive,
}

var addToSchemes = runtime.SchemeBuilder{
	qstsv1a1.AddToScheme,
}

var mutatingHookFuncs = []func(*zap.SugaredLogger, *config.Config) *webhook.OperatorWebhook{
	quarksstatefulset.NewQuarksStatefulSetPodMutator,
	statefulset.NewStatefulSetRolloutMutator,
}

// AddToManager adds all Controllers to the Manager
func AddToManager(ctx context.Context, config *config.Config, m manager.Manager) error {
	for _, f := range addToManagerFuncs {
		if err := f(ctx, config, m); err != nil {
			return err
		}
	}
	return nil
}

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return addToSchemes.AddToScheme(s)
}

// AddHooks adds all web hooks to the Manager
func AddHooks(ctx context.Context, config *config.Config, m manager.Manager, generator credsgen.Generator) error {
	ctxlog.Infof(ctx, "Setting up webhook server on %s:%d", config.WebhookServerHost, config.WebhookServerPort)

	webhookConfig := webhook.NewConfig(m.GetClient(), config, generator, WebhookConfigPrefix+config.OperatorNamespace)

	hookServer := m.GetWebhookServer()
	hookServer.CertDir = webhookConfig.CertDir

	hookServer.Register(HTTPReadyzEndpoint, ordinaryHTTPHandler())

	log := ctxlog.ExtractLogger(ctx)

	mutatingWebhooks := make([]*webhook.OperatorWebhook, len(mutatingHookFuncs))
	for idx, f := range mutatingHookFuncs {
		hook := f(log, config)
		mutatingWebhooks[idx] = hook
		hookServer.Register(hook.Path, hook.Webhook)
	}

	ctxlog.Info(ctx, "Generating webhook certificates")
	err := webhookConfig.SetupCertificate(ctx, "qsts-webhook")
	if err != nil {
		return errors.Wrap(err, "setting up the webhook server certificate")
	}

	ctxlog.Info(ctx, "Generating mutating webhook server configuration")
	err = webhookConfig.CreateMutationWebhookServerConfig(ctx, "qsts-webhook", mutatingWebhooks)
	if err != nil {
		return errors.Wrap(err, "generating the webhook server configuration")
	}

	return nil
}

func ordinaryHTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
