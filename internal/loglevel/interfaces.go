package loglevel

import (
	"context"
	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-registry-lib/config"
)

type DoguConfigRepository interface {
	Get(context.Context, dogu.SimpleName) (config.DoguConfig, error)
	Update(context.Context, config.DoguConfig) (config.DoguConfig, error)
}

type DoguRestarter interface {
	Restart(ctx context.Context, doguName string) error
}

type DoguDescriptorGetter interface {
	GetCurrent(ctx context.Context, simpleDoguName string) (*core.Dogu, error)
}

type LogLevelHandler interface {
	GetLogLevel(ctx context.Context, element any) (LogLevel, error)
	SetLogLevel(ctx context.Context, element any, targetLogLevel LogLevel) error
	Restart(ctx context.Context, name string) error
	Kind() string
}

type DoguRestartInterface interface {
	doguClient.DoguRestartInterface
}
