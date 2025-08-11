package controller

import (
	"github.com/cloudogu/ces-commons-lib/dogu"
	componentClient "github.com/cloudogu/k8s-component-operator/pkg/api/ecosystem"
	libclient "github.com/cloudogu/k8s-debug-mode-cr-lib/pkg/client/v1"
	"github.com/cloudogu/k8s-debug-mode-operator/internal/loglevel"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	typev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

//nolint:unused
//goland:noinspection GoUnusedType
type controllerManager interface {
	ctrl.Manager
}

//nolint:unused
//goland:noinspection GoUnusedType
type debugModeInterface interface {
	libclient.DebugModeInterface
}

type doguInterface interface {
	doguClient.DoguInterface
}

type debugModeV1Interface interface {
	libclient.DebugModeV1Interface
}

type doguVersionRegistry interface {
	dogu.VersionRegistry
}

type localDoguDescriptorRepository interface {
	dogu.LocalDoguDescriptorRepository
}

type componentInterface interface {
	componentClient.ComponentInterface
}

type configurationMap interface {
	typev1.ConfigMapInterface
}

type LogLevelHandler interface {
	loglevel.LogLevelHandler
}
