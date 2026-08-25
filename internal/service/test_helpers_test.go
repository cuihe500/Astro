package service

import (
	"context"

	"github.com/cuihe500/astro/internal/k8s"
)

type fakeAppAdapter struct {
	ensureNamespaceErr  error
	deleteNamespaceErr  error
	createAppErr        error
	deleteAppErr        error
	createAppFunc       func(k8s.AppSpec) error
	ensuredNamespace    string
	createdSpec         k8s.AppSpec
	deleteNamespaceCtx  context.Context
	deleteAppCtx        context.Context
	deletedNamespace    string
	deletedAppName      string
	deletedAppNamespace string
}

func (a *fakeAppAdapter) EnsureNamespace(_ context.Context, namespace string) error {
	a.ensuredNamespace = namespace
	return a.ensureNamespaceErr
}

func (a *fakeAppAdapter) DeleteNamespace(ctx context.Context, namespace string) error {
	a.deleteNamespaceCtx = ctx
	a.deletedNamespace = namespace
	return a.deleteNamespaceErr
}

func (a *fakeAppAdapter) CreateApp(_ context.Context, spec k8s.AppSpec) error {
	a.createdSpec = spec
	if a.createAppFunc != nil {
		return a.createAppFunc(spec)
	}
	return a.createAppErr
}

func (a *fakeAppAdapter) DeleteApp(ctx context.Context, name, namespace string) error {
	a.deleteAppCtx = ctx
	a.deletedAppName = name
	a.deletedAppNamespace = namespace
	return a.deleteAppErr
}

func (a *fakeAppAdapter) ScaleApp(context.Context, string, string, int32) error { return nil }

func (a *fakeAppAdapter) GetAppStatus(context.Context, string, string) (*k8s.AppStatus, error) {
	return &k8s.AppStatus{Status: "stopped"}, nil
}

func (a *fakeAppAdapter) RestartApp(context.Context, string, string) error { return nil }

func (a *fakeAppAdapter) GetAppLogs(context.Context, string, string, int64) (string, error) {
	return "", nil
}
