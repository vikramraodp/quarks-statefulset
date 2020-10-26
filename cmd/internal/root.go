package cmd

import (
	golog "log"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc" // from https://github.com/kubernetes/client-go/issues/345
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"code.cloudfoundry.org/quarks-statefulset/pkg/kube/operator"
	"code.cloudfoundry.org/quarks-statefulset/version"
	"code.cloudfoundry.org/quarks-utils/pkg/cmd"
	"code.cloudfoundry.org/quarks-utils/pkg/config"
	"code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"code.cloudfoundry.org/quarks-utils/pkg/logger"
)

var (
	log *zap.SugaredLogger
)

func wrapError(err error, msg string) error {
	return errors.Wrapf(err, "quarks-statefulset command failed. %s", msg)
}

var rootCmd = &cobra.Command{
	Use:   "quarks-statefulset",
	Short: "quarks-statefulset manages statefulsets on Kubernetes",
	RunE: func(_ *cobra.Command, args []string) error {
		log = logger.NewControllerLogger(cmd.LogLevel())
		defer log.Sync()

		restConfig, err := cmd.KubeConfig(log)
		if err != nil {
			return wrapError(err, "")
		}

		cfg := config.NewDefaultConfig(afero.NewOsFs())

		cmd.Meltdown(cfg)
		cmd.OperatorNamespace(cfg, log, "quarks-statefulset-namespace")
		cmd.MonitoredID(cfg)

		log.Infof("Starting quarks-statefulset %s, monitoring namespaces labeled with '%s'", version.Version, cfg.MonitoredID)

		serviceHost := viper.GetString("operator-webhook-service-host")
		// Port on which the operator webhook kube service listens to.
		servicePort := viper.GetInt32("operator-webhook-service-port")
		useServiceRef := viper.GetBool("operator-webhook-use-service-reference")

		if serviceHost == "" && !useServiceRef {
			return wrapError(errors.New("couldn't determine webhook server"), "operator-webhook-service-host flag is not set (env variable: QUARKS_STS_WEBHOOK_SERVICE_HOST)")
		}

		cfg.WebhookServerHost = serviceHost
		cfg.WebhookServerPort = servicePort
		cfg.WebhookUseServiceRef = useServiceRef
		cfg.MaxQuarksStatefulSetWorkers = viper.GetInt("max-quarks-statefulset-workers")

		cmd.CtxTimeOut(cfg)

		ctx := ctxlog.NewParentContext(log)

		err = cmd.ApplyCRDs(ctx, operator.ApplyCRDs, restConfig)
		if err != nil {
			return wrapError(err, "Couldn't apply CRDs.")
		}

		mgr, err := operator.NewManager(ctx, cfg, restConfig, manager.Options{
			MetricsBindAddress: "0",
			LeaderElection:     false,
			Port:               int(servicePort),
			Host:               "0.0.0.0",
		})
		if err != nil {
			return wrapError(err, "Failed to create new manager.")
		}

		err = mgr.Start(signals.SetupSignalHandler())
		if err != nil {
			return wrapError(err, "Failed to start quarks-statefulset manager.")
		}
		return nil
	},
	TraverseChildren: true,
}

// NewCFOperatorCommand returns the `quarks-statefulset` command.
func NewCFOperatorCommand() *cobra.Command {
	return rootCmd
}

// Execute the root command, runs the server
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		golog.Fatal(err)
		os.Exit(1)
	}
}

func init() {
	pf := rootCmd.Flags()

	argToEnv := map[string]string{}

	cmd.OperatorNamespaceFlags(pf, argToEnv, "quarks-statefulset-namespace")
	cmd.MonitoredIDFlags(pf, argToEnv)
	cmd.CtxTimeOutFlags(pf, argToEnv)
	cmd.KubeConfigFlags(pf, argToEnv)
	cmd.LoggerFlags(pf, argToEnv)
	cmd.ApplyCRDsFlags(pf, argToEnv)
	cmd.MeltdownFlags(pf, argToEnv)

	pf.String("cluster-domain", "cluster.local", "The Kubernetes cluster domain")
	pf.Int("max-quarks-statefulset-workers", 1, "Maximum number of workers concurrently running QuarksStatefulSet controller")
	pf.StringP("operator-webhook-service-host", "w", "", "Hostname/IP under which the webhook server can be reached from the cluster")
	pf.StringP("operator-webhook-service-port", "p", "2999", "Port the webhook server listens on")
	pf.BoolP("operator-webhook-use-service-reference", "x", false, "If true the webhook service is targeted using a service reference instead of a URL")

	for _, name := range []string{
		"max-quarks-statefulset-workers",
		"operator-webhook-service-host",
		"operator-webhook-service-port",
		"operator-webhook-use-service-reference",
	} {
		viper.BindPFlag(name, pf.Lookup(name))
	}

	argToEnv["max-quarks-statefulset-workers"] = "MAX_QUARKS_STATEFULSET_WORKERS"
	argToEnv["operator-webhook-service-host"] = "QUARKS_STS_WEBHOOK_SERVICE_HOST"
	argToEnv["operator-webhook-service-port"] = "QUARKS_STS_WEBHOOK_SERVICE_PORT"
	argToEnv["operator-webhook-use-service-reference"] = "QUARKS_STS_WEBHOOK_USE_SERVICE_REFERENCE"

	// Add env variables to help
	cmd.AddEnvToUsage(rootCmd, argToEnv)

	// Do not display cmd usage and errors
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
}
