package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	k8sCRLib "github.com/cloudogu/k8s-debug-mode-cr-lib/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/k8s-debug-mode-operator/internal/loglevel"
)

const (
	reconcilerTimeoutInSec = 60
)

var (
	defLogger = ctrl.Log.WithName("debug-mode-operator")
)

// DebugModeReconciler reconciles a DebugMode object
type DebugModeReconciler struct {
	client                  debugModeV1Interface
	doguInterface           doguInterface
	componentInterface      componentInterface
	configMapInterface      configurationMap
	doguLogLevelGetter      loglevel.DoguLogLevelHandler
	componentLogLevelGetter loglevel.ComponentLogLevelHandler
}

func NewDebugModeReconciler(client debugModeV1Interface,
	doguInterface doguInterface,
	componentInterface componentInterface,
	configMapInterface configurationMap,
	doguLogLevelGetter loglevel.DoguLogLevelHandler,
	componentLogLevelGetter loglevel.ComponentLogLevelHandler) *DebugModeReconciler {
	return &DebugModeReconciler{
		client:                  client,
		doguInterface:           doguInterface,
		componentInterface:      componentInterface,
		configMapInterface:      configMapInterface,
		doguLogLevelGetter:      doguLogLevelGetter,
		componentLogLevelGetter: componentLogLevelGetter,
	}
}

// +kubebuilder:rbac:groups=k8s.cloudogu.com.k8s.cloudogu.com,resources=debugmodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.cloudogu.com.k8s.cloudogu.com,resources=debugmodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.cloudogu.com.k8s.cloudogu.com,resources=debugmodes/finalizers,verbs=update

func (r *DebugModeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	logger.Info("Starting Reconcile for DebugMode")

	debugModeClient := r.client.DebugMode(req.Namespace)

	cr, err := debugModeClient.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, fmt.Sprintf("ERROR: failed to get CR with name %s, %w", req.Name))
		return ctrl.Result{}, ctrlclient.IgnoreNotFound(err)
	}

	stateMap := NewStateMap(ctx, cr, r.configMapInterface, logger)

	var result ctrl.Result

	if r.isActive(cr) {
		result, err = r.activateDebugMode(ctx, cr, stateMap)
	} else {
		result, err = r.deActivateDebugMode(ctx, cr, stateMap)
	}

	if err != nil {
		logger.Error(err, "Reconciling failed")
		// TODO set condition failed
		return ctrl.Result{}, err
	}
	return result, nil
}

func (r *DebugModeReconciler) activateDebugMode(ctx context.Context, cr *k8sCRLib.DebugMode, stateMap *StateMap) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	logger.Info("Activate DebugMode")

	err := r.setPhase(ctx, cr, k8sCRLib.DebugModeStatusSet)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("ERROR failed to set phase %s: %w", k8sCRLib.DebugModeStatusSet, err)
	}

	change := false
	targetLevel, err := loglevel.CreateLogLevelFromString(cr.Spec.TargetLogLevel)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("ERROR: invalid target log level %s", cr.Spec.TargetLogLevel)
	}

	// Dogus
	doguList, err := r.doguInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("ERROR: Failed to list dogus: %w", err)
	}

	if doguList != nil && len(doguList.Items) > 0 {
		for _, dogu := range doguList.Items {
			key := fmt.Sprintf("dogu.%s", dogu.Name)
			logLevel, e := r.doguLogLevelGetter.GetLogLevel(ctx, dogu)
			if e != nil {
				return ctrl.Result{}, fmt.Errorf("ERROR: Failed to get LogLevel for dogu %s: %w", dogu.Name, e)
			}

			logger.Info(fmt.Sprintf("Get Loglevel for Dogu '%s': %s", dogu.Name, logLevel))

			current, _ := stateMap.compareWithStateMap(key, cr.Spec.TargetLogLevel)

			// this is the first time this dogu is checked -> store current level in configMap
			if current == "" {
				logger.Info(fmt.Sprintf("Update state map for dogu '%s': %s", dogu.Name, logLevel))
				e = stateMap.updateStateMap(key, logLevel.String())
				if e != nil {
					return ctrl.Result{}, fmt.Errorf("ERROR: failed to update configmap for %s with level: %s :%w", key, logLevel.String(), e)
				}
			}

			// current log level does not match target level
			if !strings.EqualFold(logLevel.String(), cr.Spec.TargetLogLevel) {
				change = true
				logger.Info(fmt.Sprintf("Change loglevel for '%s': from %s to %s", dogu.Name, logLevel, cr.Spec.TargetLogLevel))
				e = r.doguLogLevelGetter.SetLogLevelForDogu(ctx, dogu.Name, targetLevel)
				if e != nil {
					return ctrl.Result{}, fmt.Errorf("ERROR: failed to set log level %s for dogu: %s :%w", cr.Spec.TargetLogLevel, dogu.Name, e)
				}

				e = r.doguLogLevelGetter.RestartDogu(ctx, dogu.Name)
				if e != nil {
					return ctrl.Result{}, fmt.Errorf("ERROR: failed to restart dogu: %s :%w", dogu.Name, e)
				}
			}
		}
	}

	// Components
	componentList, err := r.componentInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("ERROR: Failed to list components: %w", err)
	}

	if componentList != nil && len(componentList.Items) > 0 {
		for _, component := range componentList.Items {
			key := fmt.Sprintf("component.%s", component.Name)
			logLevel, e := r.componentLogLevelGetter.GetLogLevelForComponent(ctx, component)
			if e != nil {
				return ctrl.Result{}, fmt.Errorf("ERROR: Failed to get LogLevel for component %s: %w", component.Name, e)
			}

			logger.Info(fmt.Sprintf("Get Loglevel for Component '%s': %s", component.Name, logLevel.String()))

			current, _ := stateMap.compareWithStateMap(key, cr.Spec.TargetLogLevel)

			// this is the first time this component is checked -> store current level in configMap
			if current == "" {
				e = stateMap.updateStateMap(key, logLevel.String())
				if e != nil {
					return ctrl.Result{}, fmt.Errorf("ERROR: failed to update configmap for %s with level: %s :%w", key, logLevel.String(), e)
				}
			}

			// current log level does not match target level
			if !strings.EqualFold(logLevel.String(), cr.Spec.TargetLogLevel) {
				change = true
				logger.Info(fmt.Sprintf("Change loglevel for '%s': from %s to %s", component.Name, logLevel, cr.Spec.TargetLogLevel))
				e = r.componentLogLevelGetter.SetLogLevel(ctx, component, targetLevel)
				if e != nil {
					return ctrl.Result{}, fmt.Errorf("ERROR: failed to set log level %s for component: %s :%w", cr.Spec.TargetLogLevel, component.Name, e)
				}
			}
		}
	}

	if change {
		// Trigger reconcile with timeout
		logger.Info(fmt.Sprintf("Change detected - reconcile in %d seconds", reconcilerTimeoutInSec))
		return ctrl.Result{RequeueAfter: reconcilerTimeoutInSec * time.Second}, nil
	}

	logger.Info(fmt.Sprintf("Done setting debug mode - reconcile at %s", cr.Spec.DeactivateTimestamp))
	err = r.setPhase(ctx, cr, k8sCRLib.DebugModeStatusWaitForRollback)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("ERROR failed to set phase %s: %w", k8sCRLib.DebugModeStatusWaitForRollback, err)
	}

	// there were no log level changes, so we wait for the debugMode to end
	return ctrl.Result{RequeueAfter: time.Until(cr.Spec.DeactivateTimestamp.Time)}, nil
}

func (r *DebugModeReconciler) deActivateDebugMode(ctx context.Context, cr *k8sCRLib.DebugMode, stateMap *StateMap) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	logger.Info("Deactivate DebugMode")

	err := r.setPhase(ctx, cr, k8sCRLib.DebugModeStatusRollback)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("ERROR failed to set phase %s: %w", k8sCRLib.DebugModeStatusSet, err)
	}

	change := false

	// Dogus
	doguList, err := r.doguInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("ERROR: Failed to list dogus: %w", err)
	}
	if doguList != nil && len(doguList.Items) > 0 {
		for _, dogu := range doguList.Items {
			key := fmt.Sprintf("dogu.%s", dogu.Name)
			logLevel, e := r.doguLogLevelGetter.GetLogLevel(ctx, dogu)
			if e != nil {
				return ctrl.Result{}, fmt.Errorf("ERROR: Failed to get LogLevel for dogu %s: %w", dogu.Name, e)
			}

			logger.Info(fmt.Sprintf("Get Loglevel for Dogu '%s': %s", dogu.Name, logLevel))

			current, _ := stateMap.compareWithStateMap(key, cr.Spec.TargetLogLevel)
			if current == "" {
				return ctrl.Result{}, fmt.Errorf("ERROR: no stored fallback loglevel for %s: %w", dogu.Name)
			}

			storedLevel, err := loglevel.CreateLogLevelFromString(current)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("ERROR: invalid stored log level %s", storedLevel)
			}

			// current log level does not match stored level
			if !strings.EqualFold(logLevel.String(), cr.Spec.TargetLogLevel) {
				change = true
				logger.Info(fmt.Sprintf("Change loglevel for '%s': from %s to %s", dogu.Name, logLevel, storedLevel))
				e = r.doguLogLevelGetter.SetLogLevelForDogu(ctx, dogu.Name, storedLevel)
				if e != nil {
					return ctrl.Result{}, fmt.Errorf("ERROR: failed to set log level %s for dogu: %s :%w", storedLevel.String(), dogu.Name, e)
				}

				e = r.doguLogLevelGetter.RestartDogu(ctx, dogu.Name)
				if e != nil {
					return ctrl.Result{}, fmt.Errorf("ERROR: failed to restart dogu: %s :%w", dogu.Name, e)
				}
			}
		}
	}

	if change {
		// Trigger reconcile with timeout
		logger.Info(fmt.Sprintf("Change detected - reconcile in %d seconds", reconcilerTimeoutInSec))
		return ctrl.Result{RequeueAfter: reconcilerTimeoutInSec * time.Second}, nil
	}

	logger.Info(fmt.Sprintf("Done unsetting debug mode - reconcile at %s", cr.Spec.DeactivateTimestamp))
	err = r.setPhase(ctx, cr, k8sCRLib.DebugModeStatusCompleted)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("ERROR failed to set phase %s: %w", k8sCRLib.DebugModeStatusCompleted, err)
	}

	// there were no log level changes, so we wait for the debugMode to end
	return ctrl.Result{}, nil
}

func (r *DebugModeReconciler) isActive(debugCR *k8sCRLib.DebugMode) bool {
	after := debugCR.Spec.DeactivateTimestamp.After(time.Now())
	defLogger.Info(fmt.Sprintf("Check if active: %s: %b", debugCR.Spec.DeactivateTimestamp, after))
	return after
}

func (r *DebugModeReconciler) setPhase(ctx context.Context, debugCR *k8sCRLib.DebugMode, phase k8sCRLib.StatusPhase) error {
	debugCR.Status.Phase = phase

	newCr, err := r.client.DebugMode(debugCR.Namespace).UpdateStatus(ctx, debugCR, metav1.UpdateOptions{})
	defLogger.Info(fmt.Sprintf("SetPhase %s to %v", phase, newCr))
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *DebugModeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sCRLib.DebugMode{}).
		Complete(r)
}
