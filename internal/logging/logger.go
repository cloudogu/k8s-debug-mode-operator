package logging

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	uberzap "go.uber.org/zap"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Logger struct {
	logger logr.Logger
}

func FromContext(ctx context.Context) Logger {
	return Logger{
		logger: logf.FromContext(ctx),
	}
}

func (l Logger) Info(msg string, keysAndValues ...any) {
	l.logger.Info(msg, keysAndValues...)
}

func (l Logger) Error(msg string, keysAndValues ...any) {
	l.logger.V(2).GetSink().Info(-2, msg, keysAndValues...)
}

func (l Logger) Debug(msg string, keysAndValues ...any) {
	l.logger.V(-1).GetSink().Info(1, msg, keysAndValues...)
}

func ConfigureLogger() {
	zapOpts := getZapOptions()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOpts)))
}

func getZapOptions() zap.Options {
	var logLevel uberzap.AtomicLevel
	envLogLevel, err := GetLogLevel()
	if err != nil {
		fmt.Printf("unable to get configured log level. using info level instead.\n  %s\n", err.Error())
		logLevel = uberzap.NewAtomicLevelAt(uberzap.InfoLevel)
	} else {
		logLevel, err = uberzap.ParseAtomicLevel(envLogLevel)
		if err != nil {
			fmt.Printf("error parsing configured log level. using info level instead.\n  %s\n", err.Error())
			logLevel = uberzap.NewAtomicLevelAt(uberzap.InfoLevel)
		}
	}

	zapOpts := zap.Options{
		Development: IsStageDevelopment(),
		Level:       logLevel,
	}
	return zapOpts
}

func IsStageDevelopment() bool {
	return true
}

func GetLogLevel() (string, error) {
	logLevel, err := getEnvVar("LOG_LEVEL")
	if err != nil {
		return "", fmt.Errorf("failed to get env var [LOG_LEVEL]: %w", err)
	}

	return logLevel, nil
}

func getEnvVar(name string) (string, error) {
	env, found := os.LookupEnv(name)
	if !found {
		return "", fmt.Errorf("environment variable %s must be set", name)
	}
	return env, nil
}
