package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/cloudogu/k8s-debug-mode-operator/internal/controller"
	"github.com/cloudogu/k8s-debug-mode-operator/internal/logging"
	"github.com/cloudogu/k8s-debug-mode-operator/internal/loglevel"
	"github.com/cloudogu/k8s-registry-lib/dogu"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	k8scloudogucomv1 "github.com/cloudogu/k8s-debug-mode-cr-lib/api/v1"
	k8scloudogucomclient "github.com/cloudogu/k8s-debug-mode-cr-lib/pkg/client"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	// +kubebuilder:scaffold:imports
)

var (
	scheme      = runtime.NewScheme()
	operatorLog = ctrl.Log.WithName("debug-mode-operator")
	metricsAddr string
	probeAddr   string
)

type controllerManager interface {
	manager.Manager
}

type ecosystemClientSet struct {
	kubernetes.Interface
	k8scloudogucomclient.DebugModeEcosystemInterface
	doguClient.EcoSystemV2Client
	corev1.ConfigMapInterface
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(k8scloudogucomv1.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme

}

func main() {
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")

	flag.Parse()

	err := startOperator()

	if err != nil {
		operatorLog.Error(err, "failed to start operator")
		os.Exit(1)
	}
}

func startOperator() error {
	logging.ConfigureLogger()

	options := getK8sManagerOptions()
	k8sManager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}

	ctx := ctrl.SetupSignalHandler()

	err = configureManager(ctx, k8sManager)
	if err != nil {
		return fmt.Errorf("unable to configure manager: %w", err)
	}

	err = startK8sManager(ctx, k8sManager)
	if err != nil {
		return fmt.Errorf("unable to start operator: %w", err)
	}

	return err
}

func createDebugModeClientSet(k8sManager manager.Manager) (k8scloudogucomclient.DebugModeEcosystemInterface, error) {
	debugModeClientSet, err := k8scloudogucomclient.NewDebugModeClientSet(k8sManager.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("ERROR: unable to create ecosystem clientset: %w", err)
	}
	return debugModeClientSet, nil
}

func createDoguClientSet(k8sManager manager.Manager) (*doguClient.EcoSystemV2Client, error) {
	doguClientSet, err := doguClient.NewForConfig(k8sManager.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create ecosystem client set: %w", err)
	}

	return doguClientSet, nil
}

func configureManager(ctx context.Context, k8sManager manager.Manager) error {
	logger := logging.FromContext(ctx)
	namespace, found := os.LookupEnv("NAMESPACE")
	if !found {
		namespace = "ecosystem"
	}
	logger.Debug(fmt.Sprintf("use namespace %s", namespace))

	k8sClientSet, err := kubernetes.NewForConfig(k8sManager.GetConfig())

	debugModeClientSet, err := createDebugModeClientSet(k8sManager)
	if err != nil {
		return fmt.Errorf("ERROR: failed to create debug mode client set: %w", err)
	}

	doguClientSet, err := createDoguClientSet(k8sManager)
	if err != nil {
		return fmt.Errorf("ERROR: failed to create dogu client set: %w", err)
	}

	configMapClient := k8sClientSet.CoreV1().ConfigMaps(namespace)

	ecoClientSet := ecosystemClientSet{
		k8sClientSet,
		debugModeClientSet,
		*doguClientSet,
		configMapClient,
	}

	v1DebugMode := ecoClientSet.DebugModeV1()
	doguConfig := repository.NewDoguConfigRepository(configMapClient)
	doguDescriptorGetter := controller.NewDoguGetter(
		dogu.NewDoguVersionRegistry(configMapClient),
		dogu.NewLocalDoguDescriptorRepository(configMapClient),
	)

	doguLogLevelGetter := loglevel.NewDoguLogLevelHandler(doguConfig, doguDescriptorGetter)

	debugModeReconciler := controller.NewDebugModeReconciler(
		v1DebugMode.DebugMode(namespace),
		ecoClientSet.Dogus(namespace),
		ecoClientSet.ConfigMapInterface,
		doguLogLevelGetter,
	)

	err = debugModeReconciler.SetupWithManager(k8sManager)
	if err != nil {
		return fmt.Errorf("unable to configure reconciler: %w", err)
	}

	// +kubebuilder:scaffold:builder
	err = addChecks(k8sManager)
	if err != nil {
		return fmt.Errorf("failed to add checks to the manager: %w", err)
	}

	return nil
}

func getK8sManagerOptions() manager.Options {
	namespace, found := os.LookupEnv("NAMESPACE")
	if !found {
		namespace = "ecosystem"
	}
	options := ctrl.Options{
		Scheme:  scheme,
		Metrics: server.Options{BindAddress: metricsAddr},
		Cache: cache.Options{ByObject: map[client.Object]cache.ByObject{
			// Restrict namespace for components only as we want to reconcile Deployments,
			// StatefulSets and DaemonSets across all namespaces.
			&k8scloudogucomv1.DebugMode{}: {Namespaces: map[string]cache.Config{
				namespace: {},
			}},
		}},
		HealthProbeBindAddress: probeAddr,
	}

	return options
}
func startK8sManager(ctx context.Context, k8sManager controllerManager) error {
	logger := log.FromContext(ctx).WithName("k8s-manager-start")
	logger.Info("starting manager")
	if err := k8sManager.Start(ctx); err != nil {
		return fmt.Errorf("ERROR: problem running manager: %w", err)
	}
	logger.Info("manager started")
	return nil
}

func addChecks(mgr manager.Manager) error {
	err := mgr.AddHealthzCheck("healthz", healthz.Ping)
	if err != nil {
		return fmt.Errorf("failed to add healthz check: %w", err)
	}

	err = mgr.AddReadyzCheck("readyz", healthz.Ping)
	if err != nil {
		return fmt.Errorf("failed to add readyz check: %w", err)
	}

	return nil
}
