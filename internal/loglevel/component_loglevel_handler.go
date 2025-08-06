package loglevel

import (
	"context"
	componentClient "github.com/cloudogu/k8s-component-operator/pkg/api/ecosystem"
	v1 "github.com/cloudogu/k8s-component-operator/pkg/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

const (
	mappedLogLevelKey   = "mainLogLevel"
	defaultMainLogLevel = "INFO"
)

type ComponentLogLevelHandler struct {
	componentInterface componentClient.ComponentInterface
}

func NewComponentLogLevelHandler(componentInterface componentClient.ComponentInterface) *ComponentLogLevelHandler {
	return &ComponentLogLevelHandler{
		componentInterface: componentInterface,
	}
}

func (r *ComponentLogLevelHandler) GetLogLevelForComponent(ctx context.Context, component v1.Component) (LogLevel, error) {
	return r.getLogLevel(component)
}

func (r *ComponentLogLevelHandler) SetLogLevel(ctx context.Context, component v1.Component, targetLogLevel LogLevel) error {
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

func (r *ComponentLogLevelHandler) getLogLevel(component v1.Component) (LogLevel, error) {
	loglevel := defaultMainLogLevel
	if component.Spec.MappedValues != nil {
		val, ok := component.Spec.MappedValues[mappedLogLevelKey]
		// If the key exists
		if ok {
			loglevel = val
		}
	}
	return CreateLogLevelFromString(loglevel)
}
