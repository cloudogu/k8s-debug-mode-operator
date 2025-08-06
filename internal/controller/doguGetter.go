package controller

import (
	"context"
	"fmt"
	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
)

type DoguGetter struct {
	versionRegistry doguVersionRegistry
	doguRepository  localDoguDescriptorRepository
}

func NewDoguGetter(versionRegistry doguVersionRegistry, doguRepository localDoguDescriptorRepository) *DoguGetter {
	return &DoguGetter{
		versionRegistry: versionRegistry,
		doguRepository:  doguRepository,
	}
}

func (r *DoguGetter) GetCurrent(ctx context.Context, simpleDoguName string) (*core.Dogu, error) {
	current, err := r.versionRegistry.GetCurrent(ctx, dogu.SimpleName(simpleDoguName))
	if err != nil {
		return nil, fmt.Errorf("failed to get current version for dogu %s: %w", simpleDoguName, err)
	}
	get, err := r.doguRepository.Get(ctx, current)
	if err != nil {
		return nil, fmt.Errorf("failed to get current dogu %s: %w", simpleDoguName, err)
	}
	return get, nil
}
func (r *DoguGetter) GetCurrentOfAll(ctx context.Context) ([]*core.Dogu, error) {
	allCurrentDoguVersions, err := r.versionRegistry.GetCurrentOfAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all current dogu versions: %w", err)
	}
	all, err := r.doguRepository.GetAll(ctx, allCurrentDoguVersions)
	if err != nil {
		return nil, fmt.Errorf("failed to get all dogus: %w", err)
	}

	var allCurrentDogus []*core.Dogu
	for _, doguSpec := range all {
		allCurrentDogus = append(allCurrentDogus, doguSpec)
	}
	return allCurrentDogus, nil
}
