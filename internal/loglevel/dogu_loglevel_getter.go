package loglevel

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/sirupsen/logrus"
)

const (
	loggingKey = "logging/root"
)

type DoguLogLevelGetter struct {
	doguConfigRepository DoguConfigRepository
	doguDescriptorGetter DoguDescriptorGetter
}

func NewDoguLogLevelGetter(doguConfigRepository DoguConfigRepository, doguDescriptorGetter DoguDescriptorGetter) *DoguLogLevelGetter {
	return &DoguLogLevelGetter{
		doguConfigRepository: doguConfigRepository,
		doguDescriptorGetter: doguDescriptorGetter,
	}
}

func (r *DoguLogLevelGetter) GetLogLevelForDogu(ctx context.Context, name string) (LogLevel, error) {
	doguConfig, err := r.doguConfigRepository.Get(ctx, dogu.SimpleName(name))
	if err != nil {
		return LevelUnknown, fmt.Errorf("ERROR: Failed to get LogLevel: %w", err)
	}

	return r.getLogLevel(ctx, name, doguConfig)
}

func (r *DoguLogLevelGetter) getLogLevel(ctx context.Context, doguName string, doguConfig config.DoguConfig) (LogLevel, error) {
	currentLogLevelStr := r.getConfigLogLevel(ctx, doguConfig)

	if currentLogLevelStr == "" {
		logrus.Debugf("config log level is empty, try to get default log level from dogu descrption")
		var err error
		currentLogLevelStr, err = r.getDefaultLogLevel(ctx, doguName)
		if err != nil {
			return LevelUnknown, fmt.Errorf("could not get default log level from dogu description: %w", err)
		}
	}

	if currentLogLevelStr == "" {
		logrus.Warnf("log level for dogu %s is neither set in config nor description", doguName)
		return LevelUnknown, nil
	}

	logrus.Debugf("current log level from dogu %s is %s", doguName, currentLogLevelStr)

	currentLogLevel, err := CreateLogLevelFromString(currentLogLevelStr)
	if err != nil {
		logrus.Warnf("invalid log level set for dogu %s: %s", doguName, currentLogLevelStr)

		return LevelUnknown, nil
	}

	return currentLogLevel, nil
}

func (r *DoguLogLevelGetter) getConfigLogLevel(_ context.Context, dConfig config.DoguConfig) string {
	configLevelStr, _ := dConfig.Get(loggingKey)

	return string(configLevelStr)
}

func (r *DoguLogLevelGetter) getDefaultLogLevel(ctx context.Context, doguName string) (string, error) {
	doguDescription, err := r.doguDescriptorGetter.GetCurrent(ctx, doguName)
	if err != nil {
		return "", fmt.Errorf("could not get dogu description for dogu %s: %w", doguName, err)
	}

	var defaultLevelStr string

	for _, cfgValue := range doguDescription.Configuration {
		if cfgValue.Name == loggingKey {
			defaultLevelStr = cfgValue.Default
			break
		}
	}

	return defaultLevelStr, nil
}
