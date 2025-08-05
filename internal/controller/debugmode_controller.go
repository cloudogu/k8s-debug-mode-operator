package controller

import (
	"context"
	"fmt"
	componentClient "github.com/cloudogu/k8s-component-operator/pkg/api/ecosystem"
	"github.com/cloudogu/k8s-debug-mode-operator/internal/loglevel"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sCRLib "github.com/cloudogu/k8s-debug-mode-cr-lib/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// DebugModeReconciler reconciles a DebugMode object
type DebugModeReconciler struct {
	client                  debugModeV1Interface
	doguInterface           doguClient.DoguInterface
	componentInterface      componentClient.ComponentInterface
	doguLogLevelGetter      loglevel.DoguLogLevelGetter
	componentLogLevelGetter loglevel.ComponentLogLevelGetter
}

func NewDebugModeReconciler(client debugModeV1Interface,
	doguInterface doguClient.DoguInterface,
	componentInterface componentClient.ComponentInterface,
	doguLogLevelGetter loglevel.DoguLogLevelGetter,
	componentLogLevelGetter loglevel.ComponentLogLevelGetter) *DebugModeReconciler {
	return &DebugModeReconciler{
		client:                  client,
		doguInterface:           doguInterface,
		componentInterface:      componentInterface,
		doguLogLevelGetter:      doguLogLevelGetter,
		componentLogLevelGetter: componentLogLevelGetter,
	}
}

// +kubebuilder:rbac:groups=k8s.cloudogu.com.k8s.cloudogu.com,resources=debugmodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.cloudogu.com.k8s.cloudogu.com,resources=debugmodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.cloudogu.com.k8s.cloudogu.com,resources=debugmodes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DebugMode object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *DebugModeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	logger.Info("Starting Reconcile for DebugMode abc")
	logger.Info("--- 0")
	debugModeClient := r.client.DebugMode(req.Namespace)
	logger.Info("--- a")

	cr, err := debugModeClient.Get(ctx, req.Name, v1.GetOptions{})
	logger.Info("--- b")
	if err != nil {
		logger.Error(err, fmt.Sprintf("ERROR: failed to get CR with name %s", req.Name))
		return ctrl.Result{}, client.IgnoreNotFound(err)

	}
	logger.Info("--- c")
	logger.Info(fmt.Sprintf("Found a CR with endtime: %s", cr.Spec.DeactivateTimestamp))
	logger.Info("--- d")

	// List all Dogus
	doguList, err := r.doguInterface.List(ctx, v1.ListOptions{})
	logger.Info("--- e")
	r.handleErrorWithReconcile(ctx, "ERROR: Failed to list dogus: %w", err)
	logger.Info(fmt.Sprintf("%v", doguList))

	if len(doguList.Items) > 0 {
		for _, dogu := range doguList.Items {
			doguLogLevel, err := r.doguLogLevelGetter.GetLogLevelForDogu(ctx, dogu.Name)
			if err != nil {
				logger.Error(err, fmt.Sprintf("ERROR: Failed to get LogLevel for dogu %s: %w", dogu.Name, err))
			}

			logger.Info(fmt.Sprintf("xxxxxxxxxxxxxxxxxxxxxxxxxxxx %s", doguLogLevel.String()))
		}
	}

	componentList, err := r.componentInterface.List(ctx, v1.ListOptions{})
	if len(componentList.Items) > 0 {
		for _, component := range componentList.Items {
			componentLogLevel, err := r.componentLogLevelGetter.GetLogLevelForComponent(ctx, component)
			if err != nil {
				logger.Error(err, fmt.Sprintf("ERROR: Failed to get LogLevel for component %s: %w", component.Name, err))
			}
			logger.Info(fmt.Sprintf("yyyyyyyyyyyyyyyyyyyyyyyyyyyyyy %s", componentLogLevel.String()))
		}
	}

	logger.Info(fmt.Sprintf("%v", doguList))

	return ctrl.Result{}, nil
}

func (r *DebugModeReconciler) handleErrorWithReconcile(ctx context.Context, msg string, err error) {
	logger := logf.FromContext(ctx)
	if err != nil {
		logger.Error(err, fmt.Sprintf(msg, err))
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *DebugModeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sCRLib.DebugMode{}).
		Complete(r)
}
