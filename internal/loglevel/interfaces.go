package loglevel

import (
	"context"
	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-registry-lib/config"
)

type DoguConfigRepository interface {
	Get(context.Context, dogu.SimpleName) (config.DoguConfig, error)
	Update(context.Context, config.DoguConfig) (config.DoguConfig, error)
}

type DoguRestarter interface {
	RestartDogu(ctx context.Context, doguName string) error
}

type DoguDescriptorGetter interface {
	GetCurrent(ctx context.Context, simpleDoguName string) (*core.Dogu, error)
}
