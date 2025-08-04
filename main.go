package main

import (
	"context"
	"flag"
	"fmt"
	componentClient "github.com/cloudogu/k8s-component-operator/pkg/api/ecosystem"
	"github.com/cloudogu/k8s-debug-mode-operator/internal/controller"
	"github.com/cloudogu/k8s-debug-mode-operator/internal/logging"
	"github.com/cloudogu/k8s-debug-mode-operator/internal/loglevel"
	"github.com/cloudogu/k8s-registry-lib/dogu"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	controllerruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"

	k8scloudogucomv1 "github.com/cloudogu/k8s-debug-mode-cr-lib/api/v1"
	k8scloudogucomclient "github.com/cloudogu/k8s-debug-mode-cr-lib/pkg/client"
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
	componentClient.V1Alpha1Client
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(k8scloudogucomv1.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

func main() {
	logging.ConfigureLogger()
	ctx := ctrl.SetupSignalHandler()
	opLogger := log.FromContext(ctx)
	opLogger.Info("1")

	err := startOperator()
	if err != nil {
		operatorLog.Error(err, "failed to start operator")
		os.Exit(1)
	}
}

func startOperator() error {
	fmt.Printf("Hello There ----------")

	restConfig := controllerruntimeconfig.GetConfigOrDie()

	options := getK8sManagerOptions()

	k8sManager, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}

	ctx := ctrl.SetupSignalHandler()

	opLogger := log.FromContext(ctx)
	opLogger.Info("1")

	k8sClientSet, err := kubernetes.NewForConfig(restConfig)
	opLogger.Info("2")
	debugModeClient, err := createDebugModeClientSet(restConfig)
	opLogger.Info("3")
	if err != nil {
		return fmt.Errorf("ERROR: failed to create debug mode client set: %w", err)
	}
	doguClientSet, err := createDoguClientSet(restConfig)
	if err != nil {
		return fmt.Errorf("ERROR: failed to create dogu client set: %w", err)
	}
	componentClientSet, err := createComponentClientSet(restConfig)
	if err != nil {
		return fmt.Errorf("ERROR: failed to create dogu client set: %w", err)
	}
	ecoClientSet := ecosystemClientSet{
		k8sClientSet,
		debugModeClient,
		*doguClientSet,
		*componentClientSet,
	}

	v1DebugMode := ecoClientSet.DebugModeV1()

	configMapClient := k8sClientSet.CoreV1().ConfigMaps("ecosystem")
	doguConfig := repository.NewDoguConfigRepository(configMapClient)
	doguDescriptorGetter := controller.NewDoguGetter(
		dogu.NewDoguVersionRegistry(configMapClient),
		dogu.NewLocalDoguDescriptorRepository(configMapClient),
	)

	doguLogLevelGetter := loglevel.NewDoguLogLevelGetter(doguConfig, doguDescriptorGetter)
	componentLogLevelGetter := loglevel.NewComponentLogLevelGetter()

	debugModeReconciler := controller.NewDebugModeReconciler(
		v1DebugMode, ecoClientSet.Dogus("ecosystem"),
		ecoClientSet.Components("ecosystem"),
		*doguLogLevelGetter,
		*componentLogLevelGetter,
	)

	err = configureManager(k8sManager, debugModeReconciler)
	if err != nil {
		return fmt.Errorf("unable to configure manager: %w", err)
	}

	err = startK8sManager(ctx, k8sManager)
	if err != nil {
		return fmt.Errorf("unable to start operator: %w", err)
	}
	return err
}

func createDebugModeClientSet(restConfig *rest.Config) (k8scloudogucomclient.DebugModeEcosystemInterface, error) {
	debugModeClientSet, err := k8scloudogucomclient.NewDebugModeClientSet(restConfig)
	if err != nil {
		return nil, fmt.Errorf("ERROR: unable to create ecosystem clientset: %w", err)
	}
	return debugModeClientSet, nil
}

func createDoguClientSet(config *rest.Config) (*doguClient.EcoSystemV2Client, error) {
	ecosystemClientSet, err := doguClient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ecosystem client set: %w", err)
	}

	return ecosystemClientSet, nil
}

func createComponentClientSet(config *rest.Config) (*componentClient.V1Alpha1Client, error) {
	ecosystemClientSet, err := componentClient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ecosystem client set: %w", err)
	}

	return ecosystemClientSet, nil
}

func configureManager(k8sManager controllerManager, debugModeReconciler *controller.DebugModeReconciler) error {
	err := debugModeReconciler.SetupWithManager(k8sManager)
	if err != nil {
		return fmt.Errorf("unable to configure reconciler: %w", err)
	}
	return nil
}

func getK8sManagerOptions() manager.Options {
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")

	options := ctrl.Options{
		Scheme:  scheme,
		Metrics: server.Options{BindAddress: metricsAddr},
		Cache: cache.Options{ByObject: map[client.Object]cache.ByObject{
			// Restrict namespace for components only as we want to reconcile Deployments,
			// StatefulSets and DaemonSets across all namespaces.
			&k8scloudogucomv1.DebugMode{}: {Namespaces: map[string]cache.Config{
				"ecosystem": {},
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

	return nil
}
