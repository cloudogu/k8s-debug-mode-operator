package loglevel

import (
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-registry-lib/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	dogulib "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/require"
)

func Test_DoguLogLevelHandler_NewComponentLogLevelHandler(t *testing.T) {
	t.Run("should create new component log level handler", func(t *testing.T) {

		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)

		// then
		assert.NotEmpty(t, dllh)
	})
}

func Test_DoguLogLevelHandler_Kind(t *testing.T) {
	t.Run("should get component as kind", func(t *testing.T) {

		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)

		// then
		assert.NotEmpty(t, dllh)
		assert.Equal(t, "dogu", dllh.Kind())
	})
}

func Test_DoguLogLevelHandler_GetLogLevel(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {
		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)
		dogu := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mydogu",
			},
		}
		entries := config.Entries{}
		entries[loggingKey] = "warn"
		dogucConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
			Config:   config.CreateConfig(entries),
		}

		doguConfigRepository.EXPECT().Get(ctx, dogucConfig.DoguName).Return(dogucConfig, nil)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)

		level, err := dllh.GetLogLevel(ctx, dogu)

		// then
		assert.NoError(t, err)
		assert.Equal(t, LevelWarn, level)
	})
	t.Run("error getting loglevel key from config", func(t *testing.T) {
		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)
		dogu := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mydogu",
			},
		}
		entries := config.Entries{}
		entries[loggingKey] = "warn"
		dogucConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
		}

		doguConfigRepository.EXPECT().Get(ctx, dogucConfig.DoguName).Return(dogucConfig, assert.AnError)
		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)

		level, err := dllh.GetLogLevel(ctx, dogu)

		// then
		assert.Error(t, err)
		assert.Equal(t, LevelUnknown, level)
	})
	t.Run("success with default loglevel", func(t *testing.T) {
		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)
		dogu := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mydogu",
			},
		}
		entries := config.Entries{}
		entries[loggingKey] = ""
		dogucConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
			Config:   config.CreateConfig(entries),
		}

		coreDogu := core.Dogu{
			Configuration: []core.ConfigurationField{
				{
					Name:    loggingKey,
					Default: "info",
				},
			},
		}

		doguConfigRepository.EXPECT().Get(ctx, dogucConfig.DoguName).Return(dogucConfig, nil)
		doguDescriptorGetter.EXPECT().GetCurrent(ctx, dogu.Name).Return(&coreDogu, nil)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)

		level, err := dllh.GetLogLevel(ctx, dogu)

		// then
		assert.NoError(t, err)
		assert.Equal(t, LevelInfo, level)
	})
	t.Run("error with default loglevel", func(t *testing.T) {
		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)
		dogu := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mydogu",
			},
		}
		entries := config.Entries{}
		entries[loggingKey] = ""
		dogucConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
			Config:   config.CreateConfig(entries),
		}

		doguConfigRepository.EXPECT().Get(ctx, dogucConfig.DoguName).Return(dogucConfig, nil)
		doguDescriptorGetter.EXPECT().GetCurrent(ctx, dogu.Name).Return(nil, assert.AnError)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)

		level, err := dllh.GetLogLevel(ctx, dogu)

		// then
		assert.Error(t, err)
		assert.Equal(t, LevelUnknown, level)
	})
	t.Run("success without default loglevel", func(t *testing.T) {
		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)
		dogu := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mydogu",
			},
		}
		entries := config.Entries{}
		entries[loggingKey] = ""
		dogucConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
			Config:   config.CreateConfig(entries),
		}

		coreDogu := core.Dogu{
			Configuration: []core.ConfigurationField{
				{
					Name:    loggingKey,
					Default: "",
				},
			},
		}

		doguConfigRepository.EXPECT().Get(ctx, dogucConfig.DoguName).Return(dogucConfig, nil)
		doguDescriptorGetter.EXPECT().GetCurrent(ctx, dogu.Name).Return(&coreDogu, nil)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)

		level, err := dllh.GetLogLevel(ctx, dogu)

		// then
		assert.NoError(t, err)
		assert.Equal(t, LevelUnknown, level)
	})
	t.Run("success with unknown loglevel", func(t *testing.T) {
		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)
		dogu := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mydogu",
			},
		}
		entries := config.Entries{}
		entries[loggingKey] = "invalid"
		dogucConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
			Config:   config.CreateConfig(entries),
		}

		doguConfigRepository.EXPECT().Get(ctx, dogucConfig.DoguName).Return(dogucConfig, nil)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)

		level, err := dllh.GetLogLevel(ctx, dogu)

		// then
		assert.NoError(t, err)
		assert.Equal(t, LevelUnknown, level)
	})
}

func Test_DoguLogLevelHandler_SetLogLevel(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {
		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)
		dogu := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mydogu",
			},
		}
		entries := config.Entries{}
		entries[loggingKey] = "info"
		dogucConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
			Config:   config.CreateConfig(entries),
		}
		warnentries := config.Entries{}
		warnentries[loggingKey] = "warn"
		expectedConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
			Config:   config.CreateConfig(warnentries),
		}
		var err error
		expectedConfig.Config, err = expectedConfig.Config.Set(loggingKey, "WARN")

		doguConfigRepository.EXPECT().Get(ctx, dogucConfig.DoguName).Return(dogucConfig, nil)

		doguConfigRepository.EXPECT().Update(ctx, config.DoguConfig{DoguName: dogucConfig.DoguName, Config: expectedConfig.Config}).Return(dogucConfig, nil)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)
		err = dllh.SetLogLevel(ctx, dogu, LevelWarn)

		assert.NoError(t, err)

	})
	t.Run("error getting current level", func(t *testing.T) {
		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)
		dogu := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mydogu",
			},
		}
		entries := config.Entries{}
		entries[loggingKey] = ""
		dogucConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
			Config:   config.CreateConfig(entries),
		}

		warnentries := config.Entries{}
		warnentries[loggingKey] = "warn"
		expectedConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
			Config:   config.CreateConfig(warnentries),
		}
		var err error
		expectedConfig.Config, err = expectedConfig.Config.Set(loggingKey, "WARN")

		doguConfigRepository.EXPECT().Get(ctx, dogucConfig.DoguName).Return(dogucConfig, nil)
		doguDescriptorGetter.EXPECT().GetCurrent(ctx, dogu.Name).Return(nil, assert.AnError)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)
		err = dllh.SetLogLevel(ctx, dogu, LevelWarn)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error getting current log level")
	})
	t.Run("error getting config", func(t *testing.T) {
		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)
		dogu := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mydogu",
			},
		}
		entries := config.Entries{}
		entries[loggingKey] = "info"
		dogucConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
			Config:   config.CreateConfig(entries),
		}

		doguConfigRepository.EXPECT().Get(ctx, dogucConfig.DoguName).Return(dogucConfig, assert.AnError)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)
		err := dllh.SetLogLevel(ctx, dogu, LevelWarn)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ERROR: Failed to get LogLevel")
	})
	t.Run("success no change", func(t *testing.T) {
		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)
		dogu := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mydogu",
			},
		}
		entries := config.Entries{}
		entries[loggingKey] = "info"
		dogucConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
			Config:   config.CreateConfig(entries),
		}

		doguConfigRepository.EXPECT().Get(ctx, dogucConfig.DoguName).Return(dogucConfig, nil)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)
		err := dllh.SetLogLevel(ctx, dogu, LevelInfo)

		assert.NoError(t, err)

	})
	t.Run("error setting config key", func(t *testing.T) {
		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)
		dogu := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mydogu",
			},
		}
		entries := config.Entries{}
		entries[loggingKey] = "info"
		entries[loggingKey+"/mapMe"] = "info"
		dogucConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
			Config:   config.CreateConfig(entries),
		}

		doguConfigRepository.EXPECT().Get(ctx, dogucConfig.DoguName).Return(dogucConfig, nil)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)
		err := dllh.SetLogLevel(ctx, dogu, LevelWarn)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not change log level from")
	})
	t.Run("error on update", func(t *testing.T) {
		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)
		dogu := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mydogu",
			},
		}
		entries := config.Entries{}
		entries[loggingKey] = "info"
		dogucConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
			Config:   config.CreateConfig(entries),
		}
		warnentries := config.Entries{}
		warnentries[loggingKey] = "warn"
		expectedConfig := config.DoguConfig{
			DoguName: dogulib.SimpleName(dogu.Name),
			Config:   config.CreateConfig(warnentries),
		}
		var err error
		expectedConfig.Config, err = expectedConfig.Config.Set(loggingKey, "WARN")

		doguConfigRepository.EXPECT().Get(ctx, dogucConfig.DoguName).Return(dogucConfig, nil)

		doguConfigRepository.EXPECT().Update(ctx, config.DoguConfig{DoguName: dogucConfig.DoguName, Config: expectedConfig.Config}).Return(dogucConfig, assert.AnError)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)
		err = dllh.SetLogLevel(ctx, dogu, LevelWarn)

		assert.Error(t, err)

	})
}

func Test_DoguLogLevelHandler_Restart(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {
		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)
		dogu := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mydogu",
			},
		}
		doguRestart := &v2.DoguRestart{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-", dogu.Name),
			},
			Spec: v2.DoguRestartSpec{
				DoguName: dogu.Name,
			},
		}

		doguRestartInterface.EXPECT().Create(ctx, doguRestart, metav1.CreateOptions{}).Return(doguRestart, nil)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)
		err := dllh.Restart(ctx, dogu.Name)

		assert.NoError(t, err)

	})
	t.Run("error", func(t *testing.T) {
		// given
		doguRestartInterface := NewMockDoguRestartInterface(t)
		doguConfigRepository := NewMockDoguConfigRepository(t)
		doguDescriptorGetter := NewMockDoguDescriptorGetter(t)
		dogu := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mydogu",
			},
		}
		doguRestart := &v2.DoguRestart{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: fmt.Sprintf("%s-", dogu.Name),
			},
			Spec: v2.DoguRestartSpec{
				DoguName: dogu.Name,
			},
		}

		doguRestartInterface.EXPECT().Create(ctx, doguRestart, metav1.CreateOptions{}).Return(doguRestart, assert.AnError)

		// when
		dllh := NewDoguLogLevelHandler(doguConfigRepository, doguDescriptorGetter, doguRestartInterface)
		err := dllh.Restart(ctx, dogu.Name)

		assert.Error(t, err)

	})
}
