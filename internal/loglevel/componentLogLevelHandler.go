package loglevel

import (
	"context"
	"fmt"
	v1 "github.com/cloudogu/k8s-component-operator/pkg/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

const (
	mappedLogLevelKey            = "mainLogLevel"
	defaultMainLogLevel          = "INFO"
	componentLogLevelHandlerType = "component"
)

type ComponentLogLevelHandler struct {
	componentInterface ComponentInterface
}

func NewComponentLogLevelHandler(componentInterface ComponentInterface) *ComponentLogLevelHandler {
	return &ComponentLogLevelHandler{
		componentInterface: componentInterface,
	}
}

func (r *ComponentLogLevelHandler) Kind() string {
	return componentLogLevelHandlerType
}

func (r *ComponentLogLevelHandler) GetLogLevel(ctx context.Context, element any) (LogLevel, error) {
	loglevel := defaultMainLogLevel
	component, ok := element.(v1.Component)
	if !ok {
		// Typ passt nicht
		return LevelUnknown, fmt.Errorf("unexpected type of element: %v", element)
	}
	if component.Spec.MappedValues != nil {
		val, ok := component.Spec.MappedValues[mappedLogLevelKey]
		// If the key exists
		if ok {
			loglevel = val
		}
	}
	return CreateLogLevelFromString(loglevel)
}

func (r *ComponentLogLevelHandler) SetLogLevel(ctx context.Context, element any, targetLogLevel LogLevel) error {
	component, ok := element.(v1.Component)
	if !ok {
		// Typ passt nicht
		return fmt.Errorf("unexpected type of element: %v", element)
	}
	if component.Spec.MappedValues == nil {
		component.Spec.MappedValues = map[string]string{}
	}
	component.Spec.MappedValues[mappedLogLevelKey] = strings.ToLower(targetLogLevel.String())

	_, err := r.componentInterface.Update(ctx, &component, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// No Restart needed for Component
func (s *ComponentLogLevelHandler) Restart(ctx context.Context, name string) error {
	return nil
}
