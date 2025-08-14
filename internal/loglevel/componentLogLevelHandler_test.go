package loglevel

import (
	"context"
	v1 "github.com/cloudogu/k8s-component-operator/pkg/api/v1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/require"
)

const (
	mainLogLevel = "mainLogLevel"
)

func Test_ComponentLogLevelHandler_NewComponentLogLevelHandler(t *testing.T) {
	t.Run("should create new component log level handler", func(t *testing.T) {

		// given
		componentInterface := NewMockComponentInterface(t)

		// when
		cllh := NewComponentLogLevelHandler(componentInterface)

		// then
		assert.NotEmpty(t, cllh)
	})
}

func Test_ComponentLogLevelHandler_Kind(t *testing.T) {
	t.Run("should get component as kind", func(t *testing.T) {

		// given
		componentInterface := NewMockComponentInterface(t)

		// when
		cllh := NewComponentLogLevelHandler(componentInterface)

		// then
		assert.NotEmpty(t, cllh)
		assert.Equal(t, "component", cllh.Kind())
	})
}

func Test_ComponentLogLevelHandler_GetLogLevel(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {

		// given
		componentInterface := NewMockComponentInterface(t)
		component := v1.Component{
			Spec: v1.ComponentSpec{
				MappedValues: map[string]string{
					mainLogLevel: "warn",
				},
			},
		}

		// when
		cllh := NewComponentLogLevelHandler(componentInterface)

		loglevel, err := cllh.GetLogLevel(ctx, component)

		// then
		require.NoError(t, err)
		assert.Equal(t, "WARN", loglevel.String())
	})
	t.Run("success without MappedValues", func(t *testing.T) {

		// given
		componentInterface := NewMockComponentInterface(t)
		component := v1.Component{
			Spec: v1.ComponentSpec{},
		}

		// when
		cllh := NewComponentLogLevelHandler(componentInterface)

		loglevel, err := cllh.GetLogLevel(ctx, component)

		// then
		require.NoError(t, err)
		assert.Equal(t, "INFO", loglevel.String())
	})
}

func Test_ComponentLogLevelHandler_SetLogLevel(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {
		// given
		componentInterface := NewMockComponentInterface(t)
		component := v1.Component{
			Spec: v1.ComponentSpec{
				MappedValues: map[string]string{
					mainLogLevel: "info",
				},
			},
		}
		expectedComponent := v1.Component{
			Spec: v1.ComponentSpec{
				MappedValues: map[string]string{
					mainLogLevel: "warn",
				},
			},
		}
		componentInterface.EXPECT().Update(ctx, &expectedComponent, metav1.UpdateOptions{}).RunAndReturn(
			func(ctx context.Context, comp *v1.Component, options metav1.UpdateOptions) (*v1.Component, error) {
				component.Spec.MappedValues[mainLogLevel] = comp.Spec.MappedValues[mainLogLevel]
				return comp, nil
			})

		// when
		cllh := NewComponentLogLevelHandler(componentInterface)

		err := cllh.SetLogLevel(ctx, component, LevelWarn)

		// then
		require.NoError(t, err)
		assert.Equal(t, "warn", component.Spec.MappedValues["mainLogLevel"])
	})
	t.Run("success without preset loglevel", func(t *testing.T) {
		// given
		componentInterface := NewMockComponentInterface(t)
		component := v1.Component{
			Spec: v1.ComponentSpec{},
		}
		expectedComponent := v1.Component{
			Spec: v1.ComponentSpec{
				MappedValues: map[string]string{
					mainLogLevel: "warn",
				},
			},
		}
		componentInterface.EXPECT().Update(ctx, &expectedComponent, metav1.UpdateOptions{}).RunAndReturn(
			func(ctx context.Context, comp *v1.Component, options metav1.UpdateOptions) (*v1.Component, error) {
				component.Spec.MappedValues = comp.Spec.MappedValues
				return comp, nil
			})

		// when
		cllh := NewComponentLogLevelHandler(componentInterface)

		err := cllh.SetLogLevel(ctx, component, LevelWarn)

		// then
		require.NoError(t, err)
		assert.Equal(t, "warn", component.Spec.MappedValues["mainLogLevel"])
	})
	t.Run("error in update component", func(t *testing.T) {
		// given
		componentInterface := NewMockComponentInterface(t)
		component := v1.Component{
			Spec: v1.ComponentSpec{},
		}
		expectedComponent := v1.Component{
			Spec: v1.ComponentSpec{
				MappedValues: map[string]string{
					mainLogLevel: "warn",
				},
			},
		}
		componentInterface.EXPECT().Update(ctx, &expectedComponent, metav1.UpdateOptions{}).RunAndReturn(
			func(ctx context.Context, comp *v1.Component, options metav1.UpdateOptions) (*v1.Component, error) {
				return comp, assert.AnError
			})

		// when
		cllh := NewComponentLogLevelHandler(componentInterface)

		err := cllh.SetLogLevel(ctx, component, LevelWarn)

		// then
		require.Error(t, err)
	})
}

func Test_ComponentLogLevelHandler_Restart(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {
		// given
		componentInterface := NewMockComponentInterface(t)
		// when
		cllh := NewComponentLogLevelHandler(componentInterface)
		err := cllh.Restart(ctx, "anycomponent")

		// then
		require.NoError(t, err)

	})
}
