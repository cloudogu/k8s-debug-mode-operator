package controller

import (
	v1 "github.com/cloudogu/k8s-component-operator/pkg/api/v1"
	k8sCRLib "github.com/cloudogu/k8s-debug-mode-cr-lib/api/v1"
	"github.com/cloudogu/k8s-debug-mode-operator/internal/loglevel"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"testing"
	"time"
)

func Test_DebugModeReconciler_New(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		// when
		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		// then
		assert.NotNil(t, dmc)
	})
}

func Test_DebugModeReconciler_isActive(t *testing.T) {
	t.Run("success active", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
			},
		}
		// when
		active := dmc.isActive(cr)

		// then
		assert.True(t, active)

	})
	t.Run("success inactive", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
			},
		}
		// when
		active := dmc.isActive(cr)

		// then
		assert.False(t, active)

	})
	t.Run("success no deactivationtimestamp", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		cr := &k8sCRLib.DebugMode{}
		// when
		active := dmc.isActive(cr)

		// then
		assert.False(t, active)

	})
}

func Test_DebugModeReconciler_ActivateDebugMode(t *testing.T) {
	ctx := t.Context()
	t.Run("success active", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler.EXPECT().Kind().Return("component")

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Activating Debug-Mode in progress",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelInfo, nil)

		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA": "INFO",
			},
		}

		configMapClient.EXPECT().Update(ctx, cm, metav1.UpdateOptions{}).Return(cm, nil).Once()

		// - set log level
		doguLevelHandler.EXPECT().SetLogLevel(ctx, doguList.Items[0], loglevel.LevelDebug).Return(nil)
		doguLevelHandler.EXPECT().Restart(ctx, doguList.Items[0].Name).Return(nil)

		// - doguB
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[1]).Return(loglevel.LevelWarn, nil)
		cm2 := cm.DeepCopy()
		cm2.Data["dogu.doguB"] = "WARN"
		configMapClient.EXPECT().Update(ctx, cm2, metav1.UpdateOptions{}).Return(cm2, nil).Once()

		// - set log level
		doguLevelHandler.EXPECT().SetLogLevel(ctx, doguList.Items[1], loglevel.LevelDebug).Return(nil)
		doguLevelHandler.EXPECT().Restart(ctx, doguList.Items[1].Name).Return(nil)

		// - iterate component list

		componentList := &v1.ComponentList{
			Items: []v1.Component{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		componentClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(componentList, nil)

		// - componentA
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[0]).Return(loglevel.LevelInfo, nil)
		cm3 := cm2.DeepCopy()
		cm3.Data["component.componentA"] = "INFO"

		configMapClient.EXPECT().Update(ctx, cm3, metav1.UpdateOptions{}).Return(cm3, nil).Once()

		// - set log level
		componentLevelHandler.EXPECT().SetLogLevel(ctx, componentList.Items[0], loglevel.LevelDebug).Return(nil)
		componentLevelHandler.EXPECT().Restart(ctx, componentList.Items[0].Name).Return(nil)

		// - componentB
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[1]).Return(loglevel.LevelWarn, nil)
		cm4 := cm3.DeepCopy()
		cm4.Data["component.componentB"] = "WARN"
		configMapClient.EXPECT().Update(ctx, cm4, metav1.UpdateOptions{}).Return(cm4, nil).Once()

		// - set log level
		componentLevelHandler.EXPECT().SetLogLevel(ctx, componentList.Items[1], loglevel.LevelDebug).Return(nil)
		componentLevelHandler.EXPECT().Restart(ctx, componentList.Items[1].Name).Return(nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: reconcilerTimeoutInSec * time.Second}, reconcile)
		assert.NoError(t, err)
	})
	t.Run("success active with already set debug mode", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler.EXPECT().Kind().Return("component")

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Activating Debug-Mode in progress",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState2, nil).Once()

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelDebug, nil)

		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA": "DEBUG",
			},
		}

		configMapClient.EXPECT().Update(ctx, cm, metav1.UpdateOptions{}).Return(cm, nil).Once()

		// - doguB
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[1]).Return(loglevel.LevelDebug, nil)
		cm2 := cm.DeepCopy()
		cm2.Data["dogu.doguB"] = "DEBUG"
		configMapClient.EXPECT().Update(ctx, cm2, metav1.UpdateOptions{}).Return(cm2, nil).Once()

		// - iterate component list

		componentList := &v1.ComponentList{
			Items: []v1.Component{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		componentClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(componentList, nil)

		// - componentA
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[0]).Return(loglevel.LevelDebug, nil)
		cm3 := cm2.DeepCopy()
		cm3.Data["component.componentA"] = "DEBUG"

		configMapClient.EXPECT().Update(ctx, cm3, metav1.UpdateOptions{}).Return(cm3, nil).Once()

		// - componentB
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[1]).Return(loglevel.LevelDebug, nil)
		cm4 := cm3.DeepCopy()
		cm4.Data["component.componentB"] = "DEBUG"
		configMapClient.EXPECT().Update(ctx, cm4, metav1.UpdateOptions{}).Return(cm4, nil).Once()

		// - all levels should be set to debug - so the mode is done.
		// - update condition
		crWithState3 := crWithState2.DeepCopy()
		crWithState3.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "true",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Debug-Mode set for all dogus and components",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState2, true, "Debug-Mode set for all dogus and components", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState3, nil).Once()

		crWithState4 := crWithState3.DeepCopy()
		crWithState4.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusWaitForRollback(ctx, crWithState3).Return(crWithState4, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		// the timestamp differs from run to run - it is impossible to test the correct result
		assert.NotNil(t, reconcile)
		assert.NoError(t, err)
	})
	t.Run("error setting first phase active", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)

	})
	t.Run("error setting first condition active", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Activating Debug-Mode in progress",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState2, assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)

	})
	t.Run("error creating log level from spec active", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "invalid",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Activating Debug-Mode in progress",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState2, nil)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("success active with already set debug mode", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler.EXPECT().Kind().Return("component")

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Activating Debug-Mode in progress",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState2, nil).Once()

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelDebug, nil)

		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA": "DEBUG",
			},
		}

		configMapClient.EXPECT().Update(ctx, cm, metav1.UpdateOptions{}).Return(cm, nil).Once()

		// - doguB
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[1]).Return(loglevel.LevelDebug, nil)
		cm2 := cm.DeepCopy()
		cm2.Data["dogu.doguB"] = "DEBUG"
		configMapClient.EXPECT().Update(ctx, cm2, metav1.UpdateOptions{}).Return(cm2, nil).Once()

		// - iterate component list

		componentList := &v1.ComponentList{
			Items: []v1.Component{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		componentClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(componentList, nil)

		// - componentA
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[0]).Return(loglevel.LevelDebug, nil)
		cm3 := cm2.DeepCopy()
		cm3.Data["component.componentA"] = "DEBUG"

		configMapClient.EXPECT().Update(ctx, cm3, metav1.UpdateOptions{}).Return(cm3, nil).Once()

		// - componentB
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[1]).Return(loglevel.LevelDebug, nil)
		cm4 := cm3.DeepCopy()
		cm4.Data["component.componentB"] = "DEBUG"
		configMapClient.EXPECT().Update(ctx, cm4, metav1.UpdateOptions{}).Return(cm4, nil).Once()

		// - all levels should be set to debug - so the mode is done.
		// - update condition
		crWithState3 := crWithState2.DeepCopy()
		crWithState3.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "true",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Debug-Mode set for all dogus and components",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState2, true, "Debug-Mode set for all dogus and components", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState3, assert.AnError).Once()

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)

	})
	t.Run("success active with already set debug mode", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler.EXPECT().Kind().Return("component")

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Activating Debug-Mode in progress",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState2, nil).Once()

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelDebug, nil)

		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA": "DEBUG",
			},
		}

		configMapClient.EXPECT().Update(ctx, cm, metav1.UpdateOptions{}).Return(cm, nil).Once()

		// - doguB
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[1]).Return(loglevel.LevelDebug, nil)
		cm2 := cm.DeepCopy()
		cm2.Data["dogu.doguB"] = "DEBUG"
		configMapClient.EXPECT().Update(ctx, cm2, metav1.UpdateOptions{}).Return(cm2, nil).Once()

		// - iterate component list

		componentList := &v1.ComponentList{
			Items: []v1.Component{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		componentClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(componentList, nil)

		// - componentA
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[0]).Return(loglevel.LevelDebug, nil)
		cm3 := cm2.DeepCopy()
		cm3.Data["component.componentA"] = "DEBUG"

		configMapClient.EXPECT().Update(ctx, cm3, metav1.UpdateOptions{}).Return(cm3, nil).Once()

		// - componentB
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[1]).Return(loglevel.LevelDebug, nil)
		cm4 := cm3.DeepCopy()
		cm4.Data["component.componentB"] = "DEBUG"
		configMapClient.EXPECT().Update(ctx, cm4, metav1.UpdateOptions{}).Return(cm4, nil).Once()

		// - all levels should be set to debug - so the mode is done.
		// - update condition
		crWithState3 := crWithState2.DeepCopy()
		crWithState3.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "true",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Debug-Mode set for all dogus and components",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState2, true, "Debug-Mode set for all dogus and components", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState3, nil).Once()

		crWithState4 := crWithState3.DeepCopy()
		crWithState4.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusWaitForRollback(ctx, crWithState3).Return(crWithState4, assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error getting debuglist in active", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Activating Debug-Mode in progress",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error getting loglevel on dogu active", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Activating Debug-Mode in progress",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelInfo, assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error updating statemap active", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Activating Debug-Mode in progress",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelInfo, nil)

		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA": "INFO",
			},
		}

		configMapClient.EXPECT().Update(ctx, cm, metav1.UpdateOptions{}).Return(cm, assert.AnError).Once()

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error setting loglevel active", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Activating Debug-Mode in progress",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelInfo, nil)

		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA": "INFO",
			},
		}

		configMapClient.EXPECT().Update(ctx, cm, metav1.UpdateOptions{}).Return(cm, nil).Once()

		// - set log level
		doguLevelHandler.EXPECT().SetLogLevel(ctx, doguList.Items[0], loglevel.LevelDebug).Return(assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)

	})
	t.Run("error on restart in active", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Activating Debug-Mode in progress",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelInfo, nil)

		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA": "INFO",
			},
		}

		configMapClient.EXPECT().Update(ctx, cm, metav1.UpdateOptions{}).Return(cm, nil).Once()

		// - set log level
		doguLevelHandler.EXPECT().SetLogLevel(ctx, doguList.Items[0], loglevel.LevelDebug).Return(nil)
		doguLevelHandler.EXPECT().Restart(ctx, doguList.Items[0].Name).Return(assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error getting component list in active", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Activating Debug-Mode in progress",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelInfo, nil)

		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA": "INFO",
			},
		}

		configMapClient.EXPECT().Update(ctx, cm, metav1.UpdateOptions{}).Return(cm, nil).Once()

		// - set log level
		doguLevelHandler.EXPECT().SetLogLevel(ctx, doguList.Items[0], loglevel.LevelDebug).Return(nil)
		doguLevelHandler.EXPECT().Restart(ctx, doguList.Items[0].Name).Return(nil)

		// - doguB
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[1]).Return(loglevel.LevelWarn, nil)
		cm2 := cm.DeepCopy()
		cm2.Data["dogu.doguB"] = "WARN"
		configMapClient.EXPECT().Update(ctx, cm2, metav1.UpdateOptions{}).Return(cm2, nil).Once()

		// - set log level
		doguLevelHandler.EXPECT().SetLogLevel(ctx, doguList.Items[1], loglevel.LevelDebug).Return(nil)
		doguLevelHandler.EXPECT().Restart(ctx, doguList.Items[1].Name).Return(nil)

		// - iterate component list

		componentList := &v1.ComponentList{
			Items: []v1.Component{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		componentClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(componentList, assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error getting component log level", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler.EXPECT().Kind().Return("component")

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Activating Debug-Mode in progress",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelInfo, nil)

		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA": "INFO",
			},
		}

		configMapClient.EXPECT().Update(ctx, cm, metav1.UpdateOptions{}).Return(cm, nil).Once()

		// - set log level
		doguLevelHandler.EXPECT().SetLogLevel(ctx, doguList.Items[0], loglevel.LevelDebug).Return(nil)
		doguLevelHandler.EXPECT().Restart(ctx, doguList.Items[0].Name).Return(nil)

		// - doguB
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[1]).Return(loglevel.LevelWarn, nil)
		cm2 := cm.DeepCopy()
		cm2.Data["dogu.doguB"] = "WARN"
		configMapClient.EXPECT().Update(ctx, cm2, metav1.UpdateOptions{}).Return(cm2, nil).Once()

		// - set log level
		doguLevelHandler.EXPECT().SetLogLevel(ctx, doguList.Items[1], loglevel.LevelDebug).Return(nil)
		doguLevelHandler.EXPECT().Restart(ctx, doguList.Items[1].Name).Return(nil)

		// - iterate component list

		componentList := &v1.ComponentList{
			Items: []v1.Component{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentB",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "k8s-debug-mode-operator",
						Namespace: "ecosystem",
					},
				},
			},
		}

		componentClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(componentList, nil)

		// - componentA
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[0]).Return(loglevel.LevelInfo, assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)

	})
	t.Run("success skip self", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler.EXPECT().Kind().Return("component")

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusDebugModeSet(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusSet),
				Message:            "Activating Debug-Mode in progress",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Activating Debug-Mode in progress", string(k8sCRLib.DebugModeStatusSet)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelInfo, nil)

		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA": "INFO",
			},
		}

		configMapClient.EXPECT().Update(ctx, cm, metav1.UpdateOptions{}).Return(cm, nil).Once()

		// - set log level
		doguLevelHandler.EXPECT().SetLogLevel(ctx, doguList.Items[0], loglevel.LevelDebug).Return(nil)
		doguLevelHandler.EXPECT().Restart(ctx, doguList.Items[0].Name).Return(nil)

		// - doguB
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[1]).Return(loglevel.LevelWarn, nil)
		cm2 := cm.DeepCopy()
		cm2.Data["dogu.doguB"] = "WARN"
		configMapClient.EXPECT().Update(ctx, cm2, metav1.UpdateOptions{}).Return(cm2, nil).Once()

		// - set log level
		doguLevelHandler.EXPECT().SetLogLevel(ctx, doguList.Items[1], loglevel.LevelDebug).Return(nil)
		doguLevelHandler.EXPECT().Restart(ctx, doguList.Items[1].Name).Return(nil)

		// - iterate component list

		componentList := &v1.ComponentList{
			Items: []v1.Component{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentB",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "k8s-debug-mode-operator",
						Namespace: "ecosystem",
					},
				},
			},
		}

		componentClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(componentList, nil)

		// - componentA
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[0]).Return(loglevel.LevelInfo, nil)
		cm3 := cm2.DeepCopy()
		cm3.Data["component.componentA"] = "INFO"

		configMapClient.EXPECT().Update(ctx, cm3, metav1.UpdateOptions{}).Return(cm3, nil).Once()

		// - set log level
		componentLevelHandler.EXPECT().SetLogLevel(ctx, componentList.Items[0], loglevel.LevelDebug).Return(nil)
		componentLevelHandler.EXPECT().Restart(ctx, componentList.Items[0].Name).Return(nil)

		// - componentB
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[1]).Return(loglevel.LevelWarn, nil)
		cm4 := cm3.DeepCopy()
		cm4.Data["component.componentB"] = "WARN"
		configMapClient.EXPECT().Update(ctx, cm4, metav1.UpdateOptions{}).Return(cm4, nil).Once()

		// - set log level
		componentLevelHandler.EXPECT().SetLogLevel(ctx, componentList.Items[1], loglevel.LevelDebug).Return(nil)
		componentLevelHandler.EXPECT().Restart(ctx, componentList.Items[1].Name).Return(nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: reconcilerTimeoutInSec * time.Second}, reconcile)
		assert.NoError(t, err)
	})
}

func Test_DebugModeReconciler_DeactivateDebugMode(t *testing.T) {
	ctx := t.Context()
	t.Run("success deactive", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler.EXPECT().Kind().Return("component")

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA":           "INFO",
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusRollback),
				Message:            "Deactivating Debug-Mode in progres",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Deactivating Debug-Mode in progres", string(k8sCRLib.DebugModeStatusRollback)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelDebug, nil).Once()

		// - set log level
		doguLevelHandler.EXPECT().SetLogLevel(ctx, doguList.Items[0], loglevel.LevelInfo).Return(nil).Once()
		doguLevelHandler.EXPECT().Restart(ctx, doguList.Items[0].Name).Return(nil).Once()

		// - doguB
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[1]).Return(loglevel.LevelDebug, nil)

		// - set log level
		doguLevelHandler.EXPECT().SetLogLevel(ctx, doguList.Items[1], loglevel.LevelWarn).Return(nil)
		doguLevelHandler.EXPECT().Restart(ctx, doguList.Items[1].Name).Return(nil)

		// - iterate component list

		componentList := &v1.ComponentList{
			Items: []v1.Component{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		componentClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(componentList, nil)

		// - componentA
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[0]).Return(loglevel.LevelDebug, nil)

		// - set log level
		componentLevelHandler.EXPECT().SetLogLevel(ctx, componentList.Items[0], loglevel.LevelInfo).Return(nil)
		componentLevelHandler.EXPECT().Restart(ctx, componentList.Items[0].Name).Return(nil)

		// - componentB
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[1]).Return(loglevel.LevelDebug, nil)

		// - set log level
		componentLevelHandler.EXPECT().SetLogLevel(ctx, componentList.Items[1], loglevel.LevelError).Return(nil)
		componentLevelHandler.EXPECT().Restart(ctx, componentList.Items[1].Name).Return(nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: reconcilerTimeoutInSec * time.Second}, reconcile)
		assert.NoError(t, err)
	})
	t.Run("success and complete deactive", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler.EXPECT().Kind().Return("component")

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA":           "INFO",
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusRollback),
				Message:            "Deactivating Debug-Mode in progres",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Deactivating Debug-Mode in progres", string(k8sCRLib.DebugModeStatusRollback)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelInfo, nil).Once()

		// - doguB
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[1]).Return(loglevel.LevelWarn, nil)

		// - iterate component list

		componentList := &v1.ComponentList{
			Items: []v1.Component{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		componentClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(componentList, nil)

		// - componentA
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[0]).Return(loglevel.LevelInfo, nil)

		// - componentB
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[1]).Return(loglevel.LevelError, nil)

		// - delete config map
		configMapClient.EXPECT().Delete(ctx, "debugmode-state", metav1.DeleteOptions{}).Return(nil)

		// - all levels should be set to normal - so the mode is done.
		// - update condition
		crWithState3 := crWithState2.DeepCopy()
		crWithState3.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusCompleted),
				Message:            "Debug-Mode deactivated",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState2, false, "Debug-Mode deactivated", string(k8sCRLib.DebugModeStatusCompleted)).Return(crWithState3, nil).Once()

		crWithState4 := crWithState3.DeepCopy()
		crWithState4.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusCompleted(ctx, crWithState3).Return(crWithState4, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{}, reconcile)
		assert.NoError(t, err)
	})
	t.Run("error getting loglevel deactive", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA":           "INFO",
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("success deactive", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA":           "INFO",
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusRollback),
				Message:            "Deactivating Debug-Mode in progres",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Deactivating Debug-Mode in progres", string(k8sCRLib.DebugModeStatusRollback)).Return(crWithState2, assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error getting loglevel from string deactive", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "invalid",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA":           "INFO",
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusRollback),
				Message:            "Deactivating Debug-Mode in progres",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Deactivating Debug-Mode in progres", string(k8sCRLib.DebugModeStatusRollback)).Return(crWithState2, nil)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)

	})
	t.Run("error iterating dogulist deactive", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA":           "INFO",
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusRollback),
				Message:            "Deactivating Debug-Mode in progres",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Deactivating Debug-Mode in progres", string(k8sCRLib.DebugModeStatusRollback)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error gettign loglevel deactive", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA":           "INFO",
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusRollback),
				Message:            "Deactivating Debug-Mode in progres",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Deactivating Debug-Mode in progres", string(k8sCRLib.DebugModeStatusRollback)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelDebug, assert.AnError).Once()

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error no stored log level after debug mode", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusRollback),
				Message:            "Deactivating Debug-Mode in progres",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Deactivating Debug-Mode in progres", string(k8sCRLib.DebugModeStatusRollback)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelDebug, nil).Once()

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("success deactive", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA":           "INVALID",
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusRollback),
				Message:            "Deactivating Debug-Mode in progres",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Deactivating Debug-Mode in progres", string(k8sCRLib.DebugModeStatusRollback)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelDebug, nil).Once()

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error reseting loglevel deactive", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA":           "INFO",
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusRollback),
				Message:            "Deactivating Debug-Mode in progres",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Deactivating Debug-Mode in progres", string(k8sCRLib.DebugModeStatusRollback)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelDebug, nil).Once()

		// - set log level
		doguLevelHandler.EXPECT().SetLogLevel(ctx, doguList.Items[0], loglevel.LevelInfo).Return(assert.AnError).Once()

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error reseting dogu deactive", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA":           "INFO",
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusRollback),
				Message:            "Deactivating Debug-Mode in progres",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Deactivating Debug-Mode in progres", string(k8sCRLib.DebugModeStatusRollback)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelDebug, nil).Once()

		// - set log level
		doguLevelHandler.EXPECT().SetLogLevel(ctx, doguList.Items[0], loglevel.LevelInfo).Return(nil).Once()
		doguLevelHandler.EXPECT().Restart(ctx, doguList.Items[0].Name).Return(assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("success and complete deactive", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler.EXPECT().Kind().Return("component")

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA":           "INFO",
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusRollback),
				Message:            "Deactivating Debug-Mode in progres",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Deactivating Debug-Mode in progres", string(k8sCRLib.DebugModeStatusRollback)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelInfo, nil).Once()

		// - doguB
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[1]).Return(loglevel.LevelWarn, nil)

		// - iterate component list

		componentList := &v1.ComponentList{
			Items: []v1.Component{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		componentClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(componentList, nil)

		// - componentA
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[0]).Return(loglevel.LevelInfo, nil)

		// - componentB
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[1]).Return(loglevel.LevelError, nil)

		// - delete config map
		configMapClient.EXPECT().Delete(ctx, "debugmode-state", metav1.DeleteOptions{}).Return(assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error reseting state deactive", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler.EXPECT().Kind().Return("component")

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA":           "INFO",
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusRollback),
				Message:            "Deactivating Debug-Mode in progres",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Deactivating Debug-Mode in progres", string(k8sCRLib.DebugModeStatusRollback)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelInfo, nil).Once()

		// - doguB
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[1]).Return(loglevel.LevelWarn, nil)

		// - iterate component list

		componentList := &v1.ComponentList{
			Items: []v1.Component{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		componentClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(componentList, nil)

		// - componentA
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[0]).Return(loglevel.LevelInfo, nil)

		// - componentB
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[1]).Return(loglevel.LevelError, nil)

		// - delete config map
		configMapClient.EXPECT().Delete(ctx, "debugmode-state", metav1.DeleteOptions{}).Return(nil)

		// - all levels should be set to normal - so the mode is done.
		// - update condition
		crWithState3 := crWithState2.DeepCopy()
		crWithState3.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusCompleted),
				Message:            "Debug-Mode deactivated",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState2, false, "Debug-Mode deactivated", string(k8sCRLib.DebugModeStatusCompleted)).Return(crWithState3, assert.AnError).Once()

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error reseting state deactive", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		doguLevelHandler.EXPECT().Kind().Return("dogu")
		componentLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler.EXPECT().Kind().Return("component")

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA":           "INFO",
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusRollback),
				Message:            "Deactivating Debug-Mode in progres",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Deactivating Debug-Mode in progres", string(k8sCRLib.DebugModeStatusRollback)).Return(crWithState2, nil)

		// - iterate dogu list

		doguList := &v2.DoguList{
			Items: []v2.Dogu{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "doguB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		doguClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(doguList, nil)

		// - doguA
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[0]).Return(loglevel.LevelInfo, nil).Once()

		// - doguB
		doguLevelHandler.EXPECT().GetLogLevel(ctx, doguList.Items[1]).Return(loglevel.LevelWarn, nil)

		// - iterate component list

		componentList := &v1.ComponentList{
			Items: []v1.Component{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentA",
						Namespace: "ecosystem",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "componentB",
						Namespace: "ecosystem",
					},
				},
			},
		}

		componentClient.EXPECT().List(ctx, metav1.ListOptions{}).Return(componentList, nil)

		// - componentA
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[0]).Return(loglevel.LevelInfo, nil)

		// - componentB
		componentLevelHandler.EXPECT().GetLogLevel(ctx, componentList.Items[1]).Return(loglevel.LevelError, nil)

		// - delete config map
		configMapClient.EXPECT().Delete(ctx, "debugmode-state", metav1.DeleteOptions{}).Return(nil)

		// - all levels should be set to normal - so the mode is done.
		// - update condition
		crWithState3 := crWithState2.DeepCopy()
		crWithState3.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusCompleted),
				Message:            "Debug-Mode deactivated",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState2, false, "Debug-Mode deactivated", string(k8sCRLib.DebugModeStatusCompleted)).Return(crWithState3, nil).Once()

		crWithState4 := crWithState3.DeepCopy()
		crWithState4.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusCompleted(ctx, crWithState3).Return(crWithState4, assert.AnError)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, nil)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error getting cr", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "debug",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, assert.AnError)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)
	})
	t.Run("error on setting failed state", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		deactivationTime := time.Now().Add(-5 * time.Minute)

		cr := &k8sCRLib.DebugMode{
			Spec: k8sCRLib.DebugModeSpec{
				DeactivateTimestamp: metav1.NewTime(deactivationTime),
				TargetLogLevel:      "invalid",
			},
		}

		request := ctrl.Request{
			types.NamespacedName{
				Namespace: "ecosystem",
				Name:      "my_debug_mode",
			},
		}

		// - get cr from requst
		debugModeClient.EXPECT().Get(ctx, request.Name, metav1.GetOptions{}).Return(cr, nil)

		// - create new statemap
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "debugmode-state",
			},
			Data: map[string]string{
				"dogu.doguA":           "INFO",
				"dogu.doguB":           "WARN",
				"component.componentA": "INFO",
				"component.componentB": "ERROR",
			},
		}
		configMapClient.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		// - set status  SetDebugMode
		crWithState1 := cr.DeepCopy()
		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "SetDebugMode",
		}

		debugModeClient.EXPECT().UpdateStatusRollback(ctx, cr).Return(crWithState1, nil)

		// - update condition
		crWithState2 := crWithState1.DeepCopy()
		crWithState2.Status.Conditions = []metav1.Condition{
			{
				Type:               k8sCRLib.ConditionLogLevelSet,
				Status:             "false",
				ObservedGeneration: 0,
				LastTransitionTime: metav1.Time{},
				Reason:             string(k8sCRLib.DebugModeStatusRollback),
				Message:            "Deactivating Debug-Mode in progres",
			},
		}
		debugModeClient.EXPECT().AddOrUpdateLogLevelsSet(ctx, crWithState1, false, "Deactivating Debug-Mode in progres", string(k8sCRLib.DebugModeStatusRollback)).Return(crWithState2, nil)

		crWithState1.Status = k8sCRLib.DebugModeStatus{
			Phase: "Failed",
		}

		debugModeClient.EXPECT().UpdateStatusFailed(ctx, cr).Return(crWithState1, assert.AnError)

		// when
		reconcile, err := dmc.Reconcile(ctx, request)

		assert.Equal(t, ctrl.Result{RequeueAfter: 0}, reconcile)
		assert.Error(t, err)

	})
}

func Test_DebugModeReconciler_Setup(t *testing.T) {
	t.Run("setupwithmananger", func(t *testing.T) {
		// given
		debugModeClient := newMockDebugModeInterface(t)
		doguClient := newMockDoguInterface(t)
		componentClient := newMockComponentInterface(t)
		configMapClient := newMockConfigurationMap(t)

		doguLevelHandler := NewMockLogLevelHandler(t)
		componentLevelHandler := NewMockLogLevelHandler(t)

		dmc := NewDebugModeReconciler(
			debugModeClient,
			doguClient,
			componentClient,
			configMapClient,
			doguLevelHandler,
			componentLevelHandler,
		)

		mgr := newMockControllerManager(t)
		controllerOptions := config.Controller{
			SkipNameValidation: nil,
			RecoverPanic:       nil,
		}
		mgr.EXPECT().GetControllerOptions().Return(controllerOptions)

		mgr.EXPECT().GetScheme().Return(&runtime.Scheme{})

		err := dmc.SetupWithManager(mgr)

		assert.Error(t, err)
	})
}
