package loglevel

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	_ "github.com/stretchr/testify/require"
)

func Test_Level_CreateLogLevelFromString(t *testing.T) {
	t.Run("success create loglevel form string", func(t *testing.T) {
		levelMap := map[string]LogLevel{
			"info":  LevelInfo,
			"warn":  LevelWarn,
			"error": LevelError,
			"debug": LevelDebug,
		}

		for key, value := range levelMap {
			level, err := CreateLogLevelFromString(key)
			require.NoError(t, err)
			assert.Equal(t, value, level)
		}

		level, err := CreateLogLevelFromString("invalid")
		require.Error(t, err)
		assert.Equal(t, LevelUnknown.String(), level.String())

		var invalidLogLevel LogLevel = 8

		assert.Equal(t, LevelWarn.String(), invalidLogLevel.String())

	})
}
