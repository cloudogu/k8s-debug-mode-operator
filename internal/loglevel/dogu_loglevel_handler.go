package loglevel

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	loggingKey = "logging/root"
)

type DoguLogLevelHandler struct {
	doguConfigRepository DoguConfigRepository
	doguDescriptorGetter DoguDescriptorGetter
	doguRestartClient    doguClient.DoguRestartInterface
}

func NewDoguLogLevelHandler(
	doguConfigRepository DoguConfigRepository,
	doguDescriptorGetter DoguDescriptorGetter,
	doguRestartClient doguClient.DoguRestartInterface) *DoguLogLevelHandler {
	return &DoguLogLevelHandler{
		doguConfigRepository: doguConfigRepository,
		doguDescriptorGetter: doguDescriptorGetter,
		doguRestartClient:    doguRestartClient,
	}
}

func (r *DoguLogLevelHandler) GetLogLevel(ctx context.Context, d v2.Dogu) (LogLevel, error) {
	doguConfig, err := r.doguConfigRepository.Get(ctx, dogu.SimpleName(d.Name))
	if err != nil {
		return LevelUnknown, fmt.Errorf("ERROR: Failed to get LogLevel: %w", err)
	}

	return r.getLogLevel(ctx, d.Name, doguConfig)
}

func (r *DoguLogLevelHandler) SetLogLevelForDogu(ctx context.Context, name string, logLevel LogLevel) error {
	doguConfig, err := r.doguConfigRepository.Get(ctx, dogu.SimpleName(name))
	if err != nil {
		return fmt.Errorf("ERROR: Failed to get LogLevel: %w", err)
	}

	_, err = r.setLogLevel(ctx, name, doguConfig, logLevel)

	return err
}

func (r *DoguLogLevelHandler) getLogLevel(ctx context.Context, doguName string, doguConfig config.DoguConfig) (LogLevel, error) {
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

func (r *DoguLogLevelHandler) getConfigLogLevel(_ context.Context, dConfig config.DoguConfig) string {
	configLevelStr, _ := dConfig.Get(loggingKey)

	return string(configLevelStr)
}

func (r *DoguLogLevelHandler) getDefaultLogLevel(ctx context.Context, doguName string) (string, error) {
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

func (s *DoguLogLevelHandler) setLogLevel(ctx context.Context, doguName string, doguConfig config.DoguConfig, l LogLevel) (bool, error) {

	currentLogLevel, err := s.getLogLevel(ctx, doguName, doguConfig)
	if err != nil {
		return false, fmt.Errorf("Error getting current log level %s: %w", doguName, err)
	}

	if currentLogLevel == l {
		return false, nil
	}

	if lErr := s.writeLogLevel(ctx, doguConfig, l); lErr != nil {
		return false, fmt.Errorf("could not change log level from %s to %s: %w", currentLogLevel, l.String(), err)
	}

	logrus.Debugf("written new log level %s for dogu %s", l.String(), doguName)

	return true, nil
}

func (s *DoguLogLevelHandler) writeLogLevel(ctx context.Context, dConfig config.DoguConfig, l LogLevel) error {
	doguConfig, err := dConfig.Set(loggingKey, config.Value(l.String()))
	if err != nil {
		return fmt.Errorf("could not write to dogu config: %w", err)
	}

	dConfig, err = s.doguConfigRepository.Update(ctx, config.DoguConfig{DoguName: dConfig.DoguName, Config: doguConfig})
	if err != nil {
		return fmt.Errorf("could not update dogu config for dogu %q: %w", dConfig.DoguName, err)
	}

	return nil
}

// RestartDogu restarts the specified dogu.
func (s *DoguLogLevelHandler) RestartDogu(ctx context.Context, doguName string) error {
	doguRestart := &v2.DoguRestart{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", doguName),
		},
		Spec: v2.DoguRestartSpec{
			DoguName: doguName,
		},
	}
	if _, err := s.doguRestartClient.Create(ctx, doguRestart, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("failed to restart dogu %s: %w", doguName, err)
	}
	return nil
}
