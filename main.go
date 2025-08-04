package main

import (
	"context"
	"flag"
	"fmt"
	componentClient "github.com/cloudogu/k8s-component-operator/pkg/api/ecosystem"
	"github.com/cloudogu/k8s-debug-mode-operator/internal/controller"
	"github.com/cloudogu/k8s-debug-mode-operator/internal/loglevel"
	"github.com/cloudogu/k8s-registry-lib/dogu"
	"github.com/cloudogu/k8s-registry-lib/repository"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	controllerruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	k8scloudogucomv1 "github.com/cloudogu/k8s-debug-mode-cr-lib/api/v1"
	k8scloudogucomclient "github.com/cloudogu/k8s-debug-mode-cr-lib/pkg/client"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
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

// nolint:gocyclo
func main() {
	var enableLeaderElection bool
	var probeAddr string

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "f94b25f5.k8s.cloudogu.com",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	setupLog.Info("starting manager")

	ctx := ctrl.SetupSignalHandler()
	restConfig := controllerruntimeconfig.GetConfigOrDie()

	err = startOperator(ctx, mgr, restConfig)
	if err != nil {
		setupLog.Error(err, "unable to start operator")
		os.Exit(1)
	}

}

func startOperator(
	ctx context.Context,
	k8sManager ctrl.Manager,
	restConfig *rest.Config,
) error {
	k8sClientSet, err := kubernetes.NewForConfig(restConfig)

	debugModeClient, err := createDebugModeClientSet(restConfig)
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

func startK8sManager(ctx context.Context, k8sManager controllerManager) error {
	logger := log.FromContext(ctx).WithName("k8s-manager-start")
	logger.Info("starting manager")
	if err := k8sManager.Start(ctx); err != nil {
		return fmt.Errorf("ERROR: problem running manager: %w", err)
	}

	return nil
}
