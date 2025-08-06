package loglevel

import (
	"context"
	v1 "github.com/cloudogu/k8s-component-operator/pkg/api/v1"
)

const (
	mappedLogLevelKey   = "mainLogLevel"
	defaultMainLogLevel = "INFO"
)

type ComponentLogLevelGetter struct {
}

func NewComponentLogLevelGetter() *ComponentLogLevelGetter {
	return &ComponentLogLevelGetter{}
}

func (r *ComponentLogLevelGetter) GetLogLevelForComponent(ctx context.Context, component v1.Component) (LogLevel, error) {
	return r.getLogLevel(ctx, component)
}

func (r *ComponentLogLevelGetter) getLogLevel(ctx context.Context, component v1.Component) (LogLevel, error) {
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
