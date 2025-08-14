package controller

import (
	"context"
	k8sCRLib "github.com/cloudogu/k8s-debug-mode-cr-lib/api/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
)

func Test_StateMap_New(t *testing.T) {
	ctx := t.Context()
	t.Run("success with exists", func(t *testing.T) {
		//given
		cr := &k8sCRLib.DebugMode{}
		cm := &corev1.ConfigMap{}
		configMapInterface := newMockConfigurationMap(t)
		configMapInterface.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)

		//when
		stateMap := NewStateMap(ctx, cr, configMapInterface)

		//then
		assert.NotEmpty(t, stateMap)
		assert.NotNil(t, stateMap.configMap)
	})
	t.Run("success without exists", func(t *testing.T) {
		//given
		cr := &k8sCRLib.DebugMode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mycr",
				Namespace: "ecosystem"},
		}
		cm := &corev1.ConfigMap{}
		configMapInterface := newMockConfigurationMap(t)
		err := errors.NewNotFound(schema.GroupResource{}, "CM")
		configMapInterface.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, err)
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "debugmode-state",
				Namespace: cr.Namespace,
				Labels: map[string]string{
					"debugmode.k8s.cloudogu.com/owner": cr.Name,
				},
			},
			Data: map[string]string{},
		}
		configMapInterface.EXPECT().Create(ctx, cm, metav1.CreateOptions{}).Return(cm, nil)

		//when
		stateMap := NewStateMap(ctx, cr, configMapInterface)

		//then
		assert.NotEmpty(t, stateMap)
		assert.NotNil(t, stateMap.configMap)
	})
	t.Run("error without exists", func(t *testing.T) {
		//given
		cr := &k8sCRLib.DebugMode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mycr",
				Namespace: "ecosystem"},
		}
		cm := &corev1.ConfigMap{}
		configMapInterface := newMockConfigurationMap(t)
		err := assert.AnError
		configMapInterface.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, err)

		//when
		stateMap := NewStateMap(ctx, cr, configMapInterface)

		//then
		assert.NotEmpty(t, stateMap)
		assert.Nil(t, stateMap.configMap)
	})
	t.Run("error creating new configmap", func(t *testing.T) {
		//given
		cr := &k8sCRLib.DebugMode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mycr",
				Namespace: "ecosystem"},
		}
		cm := &corev1.ConfigMap{}
		configMapInterface := newMockConfigurationMap(t)
		err := errors.NewNotFound(schema.GroupResource{}, "CM")
		configMapInterface.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, err)
		cm = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "debugmode-state",
				Namespace: cr.Namespace,
				Labels: map[string]string{
					"debugmode.k8s.cloudogu.com/owner": cr.Name,
				},
			},
			Data: map[string]string{},
		}
		configMapInterface.EXPECT().Create(ctx, cm, metav1.CreateOptions{}).Return(cm, assert.AnError)

		//when
		stateMap := NewStateMap(ctx, cr, configMapInterface)

		//then
		assert.NotEmpty(t, stateMap)
		assert.Nil(t, stateMap.configMap)
	})
}

func Test_StateMap_Destroy(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {
		// given
		cm := &corev1.ConfigMap{}
		configMapInterface := newMockConfigurationMap(t)
		configMapInterface.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)
		statMap := StateMap{
			configMapInterface: configMapInterface,
		}
		configMapInterface.EXPECT().Delete(ctx, "debugmode-state", metav1.DeleteOptions{}).Return(nil)

		// when
		del, err := statMap.Destroy(ctx)

		// then
		assert.True(t, del)
		assert.NoError(t, err)
	})
	t.Run("error on get", func(t *testing.T) {
		// given
		cm := &corev1.ConfigMap{}
		configMapInterface := newMockConfigurationMap(t)
		configMapInterface.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, assert.AnError)
		statMap := StateMap{
			configMapInterface: configMapInterface,
		}

		// when
		del, err := statMap.Destroy(ctx)

		// then
		assert.False(t, del)
		assert.Error(t, err)
	})
	t.Run("error on get - no not found", func(t *testing.T) {
		// given
		cm := &corev1.ConfigMap{}
		configMapInterface := newMockConfigurationMap(t)
		notFoundErr := errors.NewNotFound(schema.GroupResource{}, "CM")
		configMapInterface.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, notFoundErr)
		statMap := StateMap{
			configMapInterface: configMapInterface,
		}

		// when
		del, err := statMap.Destroy(ctx)

		// then
		assert.False(t, del)
		assert.NoError(t, err)
	})
	t.Run("error on delete", func(t *testing.T) {
		// given
		cm := &corev1.ConfigMap{}
		configMapInterface := newMockConfigurationMap(t)
		configMapInterface.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)
		statMap := StateMap{
			configMapInterface: configMapInterface,
		}
		configMapInterface.EXPECT().Delete(ctx, "debugmode-state", metav1.DeleteOptions{}).Return(assert.AnError)

		// when
		del, err := statMap.Destroy(ctx)

		// then
		assert.False(t, del)
		assert.Error(t, err)
	})

}
func Test_StateMap_compareWithStateMap(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {
		//given
		cr := &k8sCRLib.DebugMode{}
		cm := &corev1.ConfigMap{
			Data: map[string]string{
				"dogu.keyInfo": "info",
			},
		}
		configMapInterface := newMockConfigurationMap(t)
		configMapInterface.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)
		stateMap := NewStateMap(ctx, cr, configMapInterface)

		//when
		current := stateMap.getValueFromMap("dogu.keyNotFound")
		assert.Equal(t, "", current)

		current = stateMap.getValueFromMap("dogu.keyInfo")
		assert.Equal(t, "info", current)
	})
}

func Test_StateMap_updateStateMap(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {
		//given
		cr := &k8sCRLib.DebugMode{}
		cm := &corev1.ConfigMap{
			Data: map[string]string{
				"dogu.key1": "info",
			},
		}
		configMapInterface := newMockConfigurationMap(t)
		configMapInterface.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)
		stateMap := NewStateMap(ctx, cr, configMapInterface)

		configMapInterface.EXPECT().Update(ctx, stateMap.configMap, metav1.UpdateOptions{}).RunAndReturn(
			func(ctx context.Context, configMap *corev1.ConfigMap, options metav1.UpdateOptions) (*corev1.ConfigMap, error) {
				return configMap, nil
			})

		// when
		err := stateMap.updateStateMap(ctx, "dogu.key1", "warn")

		// then
		assert.NoError(t, err)
		assert.Equal(t, "warn", stateMap.configMap.Data["dogu.key1"])
	})
	t.Run("success no data", func(t *testing.T) {
		//given
		cr := &k8sCRLib.DebugMode{}
		cm := &corev1.ConfigMap{}
		configMapInterface := newMockConfigurationMap(t)
		configMapInterface.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)
		stateMap := NewStateMap(ctx, cr, configMapInterface)

		configMapInterface.EXPECT().Update(ctx, stateMap.configMap, metav1.UpdateOptions{}).RunAndReturn(
			func(ctx context.Context, configMap *corev1.ConfigMap, options metav1.UpdateOptions) (*corev1.ConfigMap, error) {
				return configMap, nil
			})

		// when
		err := stateMap.updateStateMap(ctx, "dogu.key1", "warn")

		// then
		assert.NoError(t, err)
		assert.Equal(t, "warn", stateMap.configMap.Data["dogu.key1"])
	})
	t.Run("error on update", func(t *testing.T) {
		//given
		cr := &k8sCRLib.DebugMode{}
		cm := &corev1.ConfigMap{}
		configMapInterface := newMockConfigurationMap(t)
		configMapInterface.EXPECT().Get(ctx, "debugmode-state", metav1.GetOptions{}).Return(cm, nil)
		stateMap := NewStateMap(ctx, cr, configMapInterface)

		configMapInterface.EXPECT().Update(ctx, stateMap.configMap, metav1.UpdateOptions{}).RunAndReturn(
			func(ctx context.Context, configMap *corev1.ConfigMap, options metav1.UpdateOptions) (*corev1.ConfigMap, error) {
				return nil, assert.AnError
			})

		// when
		err := stateMap.updateStateMap(ctx, "dogu.key1", "warn")

		// then
		assert.Error(t, err)
	})
}
