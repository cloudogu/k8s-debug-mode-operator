package controller

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-debug-mode-operator/internal/logging"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"strings"
	"time"

	k8sCRLib "github.com/cloudogu/k8s-debug-mode-cr-lib/api/v1"
	"github.com/cloudogu/k8s-debug-mode-operator/internal/loglevel"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	reconcilerTimeoutInSec = 60
	phaseErrorString       = "ERROR failed to set phase %s: %w"
	conditionErrorString   = "ERROR failed to set condition %s: %w"
)

var (
	defLogger = ctrl.Log.WithName("debug-mode-operator")
)

// DebugModeReconciler reconciles a DebugMode object
type DebugModeReconciler struct {
	debugModeInterface       debugModeInterface
	doguInterface            doguInterface
	componentInterface       componentInterface
	configMapInterface       configurationMap
	doguLogLevelHandler      LogLevelHandler
	componentLogLevelHandler LogLevelHandler
}

func NewDebugModeReconciler(debugModeInterface debugModeInterface,
	doguInterface doguInterface,
	componentInterface componentInterface,
	configMapInterface configurationMap,
	doguLogLevelHandler LogLevelHandler,
	componentLogLevelHandler LogLevelHandler) *DebugModeReconciler {
	return &DebugModeReconciler{
		debugModeInterface:       debugModeInterface,
		doguInterface:            doguInterface,
		componentInterface:       componentInterface,
		configMapInterface:       configMapInterface,
		doguLogLevelHandler:      doguLogLevelHandler,
		componentLogLevelHandler: componentLogLevelHandler,
	}
}

// +kubebuilder:rbac:groups=k8s.cloudogu.com.k8s.cloudogu.com,resources=debugmodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.cloudogu.com.k8s.cloudogu.com,resources=debugmodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.cloudogu.com.k8s.cloudogu.com,resources=debugmodes/finalizers,verbs=update

func (r *DebugModeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (res ctrl.Result, err error) {
	logger := logging.FromContext(ctx)

	defer func() {
		logger.Info(fmt.Sprintf("Finished Reconcile %v : %v", res, err))
	}()

	cr, err := r.debugModeInterface.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		logger.Error(fmt.Sprintf("ERROR: failed to get CR with name %s, %v", req.Name, err))
		return ctrl.Result{}, ctrlclient.IgnoreNotFound(err)
	}
	logger.Info(fmt.Sprintf("Starting Reconcile for DebugMode: %v", cr))

	stateMap := NewStateMap(ctx, cr, r.configMapInterface)

	var result ctrl.Result

	if r.isActive(cr) {
		result, err = r.activateDebugMode(ctx, cr, stateMap)
	} else {
		result, err = r.deactivateDebugMode(ctx, cr, stateMap)
	}

	if err != nil {
		var updateerror error
		_, updateerror = r.debugModeInterface.UpdateStatusFailed(ctx, cr)
		if updateerror != nil {
			return ctrl.Result{}, updateerror
		}
		logger.Error(fmt.Sprintf("Reconciling failed: %v", err))
		return ctrl.Result{}, err
	}
	return result, nil
}

func (r *DebugModeReconciler) activateDebugMode(ctx context.Context, cr *k8sCRLib.DebugMode, stateMap *StateMap) (ctrl.Result, error) {
	logger := logging.FromContext(ctx)
	logger.Info("Activate DebugMode")

	cr, err := r.debugModeInterface.UpdateStatusDebugModeSet(ctx, cr)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(phaseErrorString, k8sCRLib.DebugModeStatusSet, err)
	}

	cr, err = r.debugModeInterface.AddOrUpdateLogLevelsSet(ctx, cr, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(conditionErrorString, k8sCRLib.DebugModeStatusSet, err)
	}

	change := false
	targetLevel, err := loglevel.CreateLogLevelFromString(cr.Spec.TargetLogLevel)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("ERROR: invalid target log level %s", cr.Spec.TargetLogLevel)
	}

	change, err = r.iterateElementsForDebugMode(ctx, true, stateMap, targetLevel, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	if change {
		// Trigger reconcile with timeout
		logger.Info(fmt.Sprintf("Change detected - reconcile in %d seconds", reconcilerTimeoutInSec))
		return ctrl.Result{RequeueAfter: reconcilerTimeoutInSec * time.Second}, nil
	}

	logger.Info(fmt.Sprintf("Done setting debug mode - reconcile at %s", cr.Spec.DeactivateTimestamp))

	cr, err = r.debugModeInterface.AddOrUpdateLogLevelsSet(ctx, cr, true, "Debug-Mode set for all dogus and components", string(k8sCRLib.DebugModeStatusSet))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(conditionErrorString, k8sCRLib.DebugModeStatusSet, err)
	}
	cr, err = r.debugModeInterface.UpdateStatusWaitForRollback(ctx, cr)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(phaseErrorString, k8sCRLib.DebugModeStatusWaitForRollback, err)
	}

	// there were no log level changes, so we wait for the debugMode to end
	return ctrl.Result{RequeueAfter: time.Until(cr.Spec.DeactivateTimestamp.Time)}, nil
}

func (r *DebugModeReconciler) activateDebugModeForElement(ctx context.Context, handler loglevel.LogLevelHandler, name string, element any, stateMap *StateMap, targetLogLevel loglevel.LogLevel, logger logging.Logger) (bool, error) {
	key := fmt.Sprintf("%s.%s", handler.Kind(), name)
	logLevel, e := handler.GetLogLevel(ctx, element)
	if e != nil {
		return false, fmt.Errorf("ERROR: Failed to get LogLevel for %s %s: %w", handler.Kind(), name, e)
	}

	current := stateMap.getValueFromMap(key)

	logger.Info(fmt.Sprintf("Loglevel for %s '%s' - current:%s - cr-state: %s", handler.Kind(), name, logLevel, current))

	// this is the first time this dogu is checked -> store current level in configMap
	if current == "" {
		logger.Info(fmt.Sprintf("Update state map for %s '%s': %s", handler.Kind(), name, logLevel))
		e = stateMap.updateStateMap(ctx, key, logLevel.String())
		if e != nil {
			return false, fmt.Errorf("ERROR: failed to update configmap for %s with level: %s :%w", key, logLevel.String(), e)
		}
	}

	// current log level does not match target level
	if !strings.EqualFold(logLevel.String(), targetLogLevel.String()) {
		logger.Info(fmt.Sprintf("Change loglevel for '%s': from %s to %s", name, logLevel, targetLogLevel.String()))
		e = handler.SetLogLevel(ctx, element, targetLogLevel)
		if e != nil {
			return false, fmt.Errorf("ERROR: failed to set log level %s for %s: %s :%w", targetLogLevel.String(), handler.Kind(), name, e)
		}

		e = handler.Restart(ctx, name)
		if e != nil {
			return false, fmt.Errorf("ERROR: failed to restart %s: %s :%w", handler.Kind(), name, e)
		}
		return true, nil
	}
	return false, nil
}

func (r *DebugModeReconciler) deactivateDebugMode(ctx context.Context, cr *k8sCRLib.DebugMode, stateMap *StateMap) (ctrl.Result, error) {
	logger := logging.FromContext(ctx)
	logger.Info("Deactivate DebugMode")

	cr, err := r.debugModeInterface.UpdateStatusRollback(ctx, cr)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(phaseErrorString, k8sCRLib.DebugModeStatusRollback, err)
	}

	cr, err = r.debugModeInterface.AddOrUpdateLogLevelsSet(ctx, cr, false, "Deactivating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusRollback))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(conditionErrorString, k8sCRLib.DebugModeStatusRollback, err)
	}

	targetLevel, err := loglevel.CreateLogLevelFromString(cr.Spec.TargetLogLevel)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("ERROR: invalid target log level %s", cr.Spec.TargetLogLevel)
	}

	change := false

	change, err = r.iterateElementsForDebugMode(ctx, false, stateMap, targetLevel, logger)
	if err != nil {
		return ctrl.Result{}, err
	}

	if change {
		// Trigger reconcile with timeout
		logger.Info(fmt.Sprintf("Change detected - reconcile in %d seconds", reconcilerTimeoutInSec))
		return ctrl.Result{RequeueAfter: reconcilerTimeoutInSec * time.Second}, nil
	}

	// the current statemap stores the values of this debugmode - if the debug mode is deactivated, the statemap is no longer needed
	destroy, err := stateMap.Destroy(ctx)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("ERROR failed to delete configmap: %w", err)
	}
	if destroy {
		logger.Debug("StateMap deleted")
	}

	logger.Info(fmt.Sprintf("Done unsetting debug mode - reconcile at %s", cr.Spec.DeactivateTimestamp))
	cr, err = r.debugModeInterface.AddOrUpdateLogLevelsSet(ctx, cr, false, "Debug-Mode deactivated", string(k8sCRLib.DebugModeStatusCompleted))
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(conditionErrorString, k8sCRLib.DebugModeStatusCompleted, err)
	}
	_, err = r.debugModeInterface.UpdateStatusCompleted(ctx, cr)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(phaseErrorString, k8sCRLib.DebugModeStatusCompleted, err)
	}

	// there were no log level changes, so we wait for the debugMode to end
	return ctrl.Result{}, nil
}

func (r *DebugModeReconciler) deactivateDebugModeForElement(ctx context.Context, handler loglevel.LogLevelHandler, name string, element any, stateMap *StateMap, logger logging.Logger) (bool, error) {
	key := fmt.Sprintf("%s.%s", handler.Kind(), name)
	logLevel, e := handler.GetLogLevel(ctx, element)
	if e != nil {
		return false, fmt.Errorf("ERROR: Failed to get LogLevel for %s %s: %w", handler.Kind(), name, e)
	}

	current := stateMap.getValueFromMap(key)

	logger.Info(fmt.Sprintf("Loglevel for %s '%s' - current:%s - cr-state: %s", handler.Kind(), name, logLevel, current))

	if current == "" {
		return false, fmt.Errorf("ERROR: no stored fallback loglevel for %s", name)
	}

	storedLevel, err := loglevel.CreateLogLevelFromString(current)
	if err != nil {
		return false, fmt.Errorf("ERROR: invalid stored log level %s", storedLevel)
	}

	// current log level does not match stored level
	if !strings.EqualFold(logLevel.String(), storedLevel.String()) {
		logger.Info(fmt.Sprintf("Change loglevel for '%s': from %s to %s", name, logLevel, storedLevel))
		e = handler.SetLogLevel(ctx, element, storedLevel)
		if e != nil {
			return false, fmt.Errorf("ERROR: failed to set log level %s for %s: %s :%w", storedLevel.String(), handler.Kind(), name, e)
		}

		e = handler.Restart(ctx, name)
		if e != nil {
			return false, fmt.Errorf("ERROR: failed to restart %s: %s :%w", handler.Kind(), name, e)
		}
		return true, nil
	}
	return false, nil
}

func (r *DebugModeReconciler) isActive(debugCR *k8sCRLib.DebugMode) bool {
	after := debugCR.Spec.DeactivateTimestamp.After(time.Now())
	defLogger.Info(fmt.Sprintf("Check if active: %s: %t", debugCR.Spec.DeactivateTimestamp, after))
	return after
}

func (r *DebugModeReconciler) iterateElementsForDebugMode(ctx context.Context, activate bool, stateMap *StateMap, targetLogLevel loglevel.LogLevel, logger logging.Logger) (bool, error) {
	doguChange, err := r.iterateDogusForDebugMode(ctx, activate, stateMap, targetLogLevel, logger)
	if err != nil {
		return doguChange, fmt.Errorf("ERROR failed to iterate dogus: %w", err)
	}

	componentChange, err := r.iterateComponentsForDebugMode(ctx, activate, stateMap, targetLogLevel, logger)
	if err != nil {
		// return doguChange or componentChange so that we don't miss changes when there's none in components
		return doguChange || componentChange, fmt.Errorf("ERROR failed to iterate components: %w", err)
	}

	return doguChange || componentChange, nil
}

func (r *DebugModeReconciler) iterateDogusForDebugMode(ctx context.Context, activate bool, stateMap *StateMap, targetLogLevel loglevel.LogLevel, logger logging.Logger) (bool, error) {
	change := false
	// Dogus
	doguList, err := r.doguInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("ERROR: Failed to list dogus: %w", err)
	}
	if doguList != nil && len(doguList.Items) > 0 {
		doguChange := false
		for _, dogu := range doguList.Items {
			if activate {
				doguChange, err = r.activateDebugModeForElement(ctx, r.doguLogLevelHandler, dogu.Name, dogu, stateMap, targetLogLevel, logger)
			} else {
				doguChange, err = r.deactivateDebugModeForElement(ctx, r.doguLogLevelHandler, dogu.Name, dogu, stateMap, logger)
			}
			change = change || doguChange
			if err != nil {
				return false, err
			}
		}
	}

	return change, nil
}

func (r *DebugModeReconciler) iterateComponentsForDebugMode(ctx context.Context, activate bool, stateMap *StateMap, targetLogLevel loglevel.LogLevel, logger logging.Logger) (bool, error) {
	change := false
	// components
	componentList, err := r.componentInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("ERROR: Failed to list components: %w", err)
	}
	if componentList != nil && len(componentList.Items) > 0 {
		componentChange := false
		for _, component := range componentList.Items {
			// skip self
			if component.Name == "k8s-debug-mode-operator" {
				continue
			}
			if activate {
				componentChange, err = r.activateDebugModeForElement(ctx, r.componentLogLevelHandler, component.Name, component, stateMap, targetLogLevel, logger)
			} else {
				componentChange, err = r.deactivateDebugModeForElement(ctx, r.componentLogLevelHandler, component.Name, component, stateMap, logger)
			}
			change = change || componentChange
			if err != nil {
				return false, err
			}
		}
	}

	return change, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DebugModeReconciler) SetupWithManager(mgr controllerManager) error {
	controllerOptions := mgr.GetControllerOptions()
	options := controller.TypedOptions[reconcile.Request]{
		SkipNameValidation: controllerOptions.SkipNameValidation,
		RecoverPanic:       controllerOptions.RecoverPanic,
	}
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithOptions(options).
		For(&k8sCRLib.DebugMode{}).
		Complete(r)
}
