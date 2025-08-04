package controller

import (
	"github.com/cloudogu/ces-commons-lib/dogu"
	libclient "github.com/cloudogu/k8s-debug-mode-cr-lib/pkg/client/v1"
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

type debugModeV1Interface interface {
	libclient.DebugModeV1Interface
}

type doguVersionRegistry interface {
	dogu.VersionRegistry
}

type localDoguDescriptorRepository interface {
	dogu.LocalDoguDescriptorRepository
}
