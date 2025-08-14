package logging

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func Test_Logger_FromContext(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {
		ConfigureLogger()
		logger := FromContext(ctx)
		assert.NotEmpty(t, logger)
	})
}

func Test_Logger_Log(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {

		os.Setenv("LOG_LEVEL", "debug")
		ConfigureLogger()
		logger := FromContext(ctx)

		logger.Info("This is Info")
		logger.Error("This is Error")
		logger.Debug("This is Debug")

		os.Setenv("LOG_LEVEL", "invalid")
		ConfigureLogger()
		logger = FromContext(ctx)
		logger.Info("This is invalid")
	})
}
