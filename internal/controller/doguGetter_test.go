package controller

import (
	dogu2 "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_DoguGetter_New(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		versionRegistry := newMockDoguVersionRegistry(t)
		doguDescritorRepo := newMockLocalDoguDescriptorRepository(t)

		// when
		doguGetter := NewDoguGetter(versionRegistry, doguDescritorRepo)

		// then
		assert.NotNil(t, doguGetter)
	})
}

func Test_DoguGetter_GetCurrent(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {
		// given
		versionRegistry := newMockDoguVersionRegistry(t)
		doguDescriptorRepo := newMockLocalDoguDescriptorRepository(t)
		doguGetter := NewDoguGetter(versionRegistry, doguDescriptorRepo)

		simpleNameVersion := dogu2.SimpleNameVersion{
			Name: "MyDogu",
			Version: core.Version{
				Raw: "1.2.3",
			},
		}

		dogu := &core.Dogu{
			Name:    "MyDogu",
			Version: "1.2.3",
		}

		versionRegistry.EXPECT().GetCurrent(ctx, dogu2.SimpleName("MyDogu")).Return(simpleNameVersion, nil)

		doguDescriptorRepo.EXPECT().Get(ctx, simpleNameVersion).Return(dogu, nil)

		// when
		retdogu, err := doguGetter.GetCurrent(ctx, "MyDogu")

		// then
		assert.NoError(t, err)
		assert.NotNil(t, retdogu)
		assert.Equal(t, dogu.Name, retdogu.Name)
	})
	t.Run("error on get current", func(t *testing.T) {
		// given
		versionRegistry := newMockDoguVersionRegistry(t)
		doguDescriptorRepo := newMockLocalDoguDescriptorRepository(t)
		doguGetter := NewDoguGetter(versionRegistry, doguDescriptorRepo)

		simpleNameVersion := dogu2.SimpleNameVersion{
			Name: "MyDogu",
			Version: core.Version{
				Raw: "1.2.3",
			},
		}

		versionRegistry.EXPECT().GetCurrent(ctx, dogu2.SimpleName("MyDogu")).Return(simpleNameVersion, assert.AnError)

		// when
		retdogu, err := doguGetter.GetCurrent(ctx, "MyDogu")

		// then
		assert.Error(t, err)
		assert.Nil(t, retdogu)
	})
	t.Run("error on get", func(t *testing.T) {
		// given
		versionRegistry := newMockDoguVersionRegistry(t)
		doguDescriptorRepo := newMockLocalDoguDescriptorRepository(t)
		doguGetter := NewDoguGetter(versionRegistry, doguDescriptorRepo)

		simpleNameVersion := dogu2.SimpleNameVersion{
			Name: "MyDogu",
			Version: core.Version{
				Raw: "1.2.3",
			},
		}

		dogu := &core.Dogu{
			Name:    "MyDogu",
			Version: "1.2.3",
		}

		versionRegistry.EXPECT().GetCurrent(ctx, dogu2.SimpleName("MyDogu")).Return(simpleNameVersion, nil)

		doguDescriptorRepo.EXPECT().Get(ctx, simpleNameVersion).Return(dogu, assert.AnError)

		// when
		retdogu, err := doguGetter.GetCurrent(ctx, "MyDogu")

		// then
		assert.Error(t, err)
		assert.Nil(t, retdogu)
	})
}

func Test_DoguGetter_GeteCurrentOfAll(t *testing.T) {
	ctx := t.Context()
	t.Run("success", func(t *testing.T) {
		// given
		versionRegistry := newMockDoguVersionRegistry(t)
		doguDescriptorRepo := newMockLocalDoguDescriptorRepository(t)
		doguGetter := NewDoguGetter(versionRegistry, doguDescriptorRepo)

		simpleNameVersion := dogu2.SimpleNameVersion{
			Name: "MyDogu",
			Version: core.Version{
				Raw: "1.2.3",
			},
		}

		dogu := &core.Dogu{
			Name:    "MyDogu",
			Version: "1.2.3",
		}

		versionRegistry.EXPECT().GetCurrentOfAll(ctx).Return([]dogu2.SimpleNameVersion{simpleNameVersion}, nil)

		doguDescriptorRepo.EXPECT().GetAll(ctx, []dogu2.SimpleNameVersion{simpleNameVersion}).Return(map[dogu2.SimpleNameVersion]*core.Dogu{
			simpleNameVersion: dogu,
		}, nil)

		// when
		retdogu, err := doguGetter.GetCurrentOfAll(ctx)

		// then
		assert.NoError(t, err)
		assert.NotNil(t, retdogu)
		assert.Equal(t, dogu.Name, retdogu[0].Name)
	})
	t.Run("error on get current of all", func(t *testing.T) {
		// given
		versionRegistry := newMockDoguVersionRegistry(t)
		doguDescriptorRepo := newMockLocalDoguDescriptorRepository(t)
		doguGetter := NewDoguGetter(versionRegistry, doguDescriptorRepo)

		simpleNameVersion := dogu2.SimpleNameVersion{
			Name: "MyDogu",
			Version: core.Version{
				Raw: "1.2.3",
			},
		}

		versionRegistry.EXPECT().GetCurrentOfAll(ctx).Return([]dogu2.SimpleNameVersion{simpleNameVersion}, assert.AnError)

		// when
		retdogu, err := doguGetter.GetCurrentOfAll(ctx)

		// then
		assert.Error(t, err)
		assert.Nil(t, retdogu)
	})
	t.Run("error on get", func(t *testing.T) {
		// given
		versionRegistry := newMockDoguVersionRegistry(t)
		doguDescriptorRepo := newMockLocalDoguDescriptorRepository(t)
		doguGetter := NewDoguGetter(versionRegistry, doguDescriptorRepo)

		simpleNameVersion := dogu2.SimpleNameVersion{
			Name: "MyDogu",
			Version: core.Version{
				Raw: "1.2.3",
			},
		}

		versionRegistry.EXPECT().GetCurrentOfAll(ctx).Return([]dogu2.SimpleNameVersion{simpleNameVersion}, nil)

		doguDescriptorRepo.EXPECT().GetAll(ctx, []dogu2.SimpleNameVersion{simpleNameVersion}).Return(nil, assert.AnError)

		// when
		retdogu, err := doguGetter.GetCurrentOfAll(ctx)

		// then
		assert.Error(t, err)
		assert.Nil(t, retdogu)
	})
}
