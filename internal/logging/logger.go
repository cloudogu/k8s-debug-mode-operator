package logging

import (
	"fmt"
	uberzap "go.uber.org/zap"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

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
