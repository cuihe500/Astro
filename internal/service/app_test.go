package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cuihe500/astro/internal/k8s"
	"github.com/cuihe500/astro/internal/model"
	"github.com/cuihe500/astro/pkg/errcode"
	"gorm.io/gorm"
)

type fakeAppStore struct {
	project       *model.Project
	app           *model.App
	appLookupErr  error
	nameLookupErr error
	createErr     error
	createCalls   int
	visible       bool
	deletedID     uint
}

func (s *fakeAppStore) CreateInProject(app *model.App, _ uint, beforeCommit func(string) error) (*model.Project, error) {
	s.createCalls++
	app.ID = 11
	if err := beforeCommit(s.project.Namespace); err != nil {
		return nil, err
	}
	if s.createErr != nil {
		return nil, s.createErr
	}
	s.visible = true
	return s.project, nil
}

func TestCreateAppValidatesReferencesBeforeTransaction(t *testing.T) {
	project := &model.Project{BaseModel: model.BaseModel{ID: 3}, UserID: 7, Namespace: "astro-project-test"}
	repo := &fakeAppStore{project: project, nameLookupErr: gorm.ErrRecordNotFound}
	adapter := &fakeAppAdapter{validateReferencesErr: errors.New("secret missing")}
	config := model.AppConfig{ImagePullSecrets: []string{"registry-secret"}}

	_, err := newAppService(repo, &fakeProjectReader{project: project}, adapter).CreateApp(
		context.Background(),
		CreateAppRequest{Name: "demo", Image: "nginx:latest", Replicas: 1, Config: config, ProjectID: 3, UserID: 7},
	)
	assertErrorCode(t, err, errcode.ErrAppCreateFailed)
	if repo.createCalls != 0 {
		t.Fatal("引用预检失败后不应启动数据库事务")
	}
	if adapter.validatedNamespace != project.Namespace || len(adapter.validatedConfig.ImagePullSecrets) != 1 {
		t.Fatal("引用预检未使用已授权项目命名空间和完整配置")
	}
}

func TestCreateAppPersistsAndMapsNormalizedConfig(t *testing.T) {
	project := &model.Project{BaseModel: model.BaseModel{ID: 3}, UserID: 7, Namespace: "astro-project-test"}
	repo := &fakeAppStore{project: project, nameLookupErr: gorm.ErrRecordNotFound}
	adapter := &fakeAppAdapter{}
	servicePort := int32(80)
	config := model.AppConfig{Ports: []model.AppPort{{
		Name: "http", ContainerPort: 8080, Protocol: "TCP", ServicePort: &servicePort,
	}}}

	app, err := newAppService(repo, &fakeProjectReader{project: project}, adapter).CreateApp(
		context.Background(),
		CreateAppRequest{Name: "demo", Image: "nginx:latest", Replicas: 1, Config: config, ProjectID: 3, UserID: 7},
	)
	if err != nil {
		t.Fatalf("创建应用失败: %v", err)
	}
	if len(app.Config.Ports) != 1 || len(adapter.createdSpec.Config.Ports) != 1 {
		t.Fatal("配置未同时写入应用模型和 Kubernetes 规格")
	}
}

func (s *fakeAppStore) Delete(id uint) error {
	s.deletedID = id
	return nil
}

func (s *fakeAppStore) GetByProjectAndID(uint, uint) (*model.App, error) {
	return s.app, s.appLookupErr
}

func (s *fakeAppStore) GetByProjectID(uint) ([]model.App, error) { return nil, nil }

func (s *fakeAppStore) GetByProjectAndName(uint, string) (*model.App, error) {
	return s.app, s.nameLookupErr
}

func (s *fakeAppStore) UpdateStatus(uint, string) error { return nil }

func (s *fakeAppStore) UpdateReplicas(uint, int) error { return nil }

type fakeProjectReader struct {
	project *model.Project
	err     error
}

func (r *fakeProjectReader) GetByID(uint) (*model.Project, error) { return r.project, r.err }

func TestCreateAppFailureRollsBackRecordAndCleansPossibleResources(t *testing.T) {
	project := &model.Project{BaseModel: model.BaseModel{ID: 3}, UserID: 7, Namespace: "astro-project-test"}
	repo := &fakeAppStore{project: project, nameLookupErr: gorm.ErrRecordNotFound}
	ctx, cancel := context.WithCancel(context.Background())
	adapter := &fakeAppAdapter{
		deleteAppErr: errors.New("kubernetes cleanup failed"),
		createAppFunc: func(k8s.AppSpec) error {
			cancel()
			return errors.New("kubernetes create failed")
		},
	}
	_, err := newAppService(repo, &fakeProjectReader{project: project}, adapter).CreateApp(
		ctx,
		CreateAppRequest{Name: "demo", Image: "nginx:latest", Replicas: 1, ProjectID: 3, UserID: 7},
	)
	assertErrorCode(t, err, errcode.ErrAppCreateFailed)
	if adapter.createdSpec.Namespace != project.Namespace {
		t.Fatalf("应用命名空间错误: got %q, want %q", adapter.createdSpec.Namespace, project.Namespace)
	}
	if repo.visible {
		t.Fatal("Kubernetes 创建失败后应用记录不应可见")
	}
	if adapter.deletedAppName != "demo" || adapter.deletedAppNamespace != project.Namespace {
		t.Fatal("Kubernetes 创建失败后未清理可能残留的资源")
	}
	if adapter.deleteAppCtx == nil || adapter.deleteAppCtx.Err() != nil {
		t.Fatal("应用补偿不应继承已取消的请求 context")
	}
	if strings.Contains(err.Error(), "数据库错误") ||
		!strings.Contains(err.Error(), "kubernetes create failed") ||
		!strings.Contains(err.Error(), "kubernetes cleanup failed") {
		t.Fatalf("组合错误未保留真实来源: %v", err)
	}
}

func TestCreateAppHidesRecordAndCompensatesKubernetesWhenCommitFails(t *testing.T) {
	project := &model.Project{BaseModel: model.BaseModel{ID: 3}, UserID: 7, Namespace: "astro-project-test"}
	repo := &fakeAppStore{
		project:       project,
		nameLookupErr: gorm.ErrRecordNotFound,
		createErr:     errors.New("commit failed"),
	}
	adapter := &fakeAppAdapter{}
	adapter.createAppFunc = func(k8s.AppSpec) error {
		if repo.visible {
			t.Fatal("事务提交前应用记录不应可见")
		}
		return nil
	}

	_, err := newAppService(repo, &fakeProjectReader{project: project}, adapter).CreateApp(
		context.Background(),
		CreateAppRequest{Name: "demo", Image: "nginx:latest", Replicas: 1, ProjectID: 3, UserID: 7},
	)
	assertErrorCode(t, err, errcode.ErrDatabase)
	if repo.visible {
		t.Fatal("事务提交失败后应用记录不应可见")
	}
	if adapter.deletedAppName != "demo" || adapter.deletedAppNamespace != project.Namespace {
		t.Fatal("事务提交失败后未清理已创建的 Kubernetes 资源")
	}
}

func TestGetAppRejectsOtherUsersProject(t *testing.T) {
	project := &model.Project{BaseModel: model.BaseModel{ID: 3}, UserID: 9}
	service := newAppService(&fakeAppStore{}, &fakeProjectReader{project: project}, &fakeAppAdapter{})
	_, err := service.GetApp(context.Background(), 3, 4, 7)
	assertErrorCode(t, err, errcode.ErrForbidden)
}

func TestCreateAppRejectsDuplicateNameInProject(t *testing.T) {
	project := &model.Project{BaseModel: model.BaseModel{ID: 3}, UserID: 7}
	service := newAppService(
		&fakeAppStore{app: &model.App{ProjectID: 3}},
		&fakeProjectReader{project: project},
		&fakeAppAdapter{},
	)
	_, err := service.CreateApp(context.Background(), CreateAppRequest{
		Name: "demo", Image: "nginx:latest", Replicas: 1, ProjectID: 3, UserID: 7,
	})
	assertErrorCode(t, err, errcode.ErrAppExists)
}

func TestGetAppRejectsAppOutsideProject(t *testing.T) {
	project := &model.Project{BaseModel: model.BaseModel{ID: 3}, UserID: 7}
	service := newAppService(
		&fakeAppStore{appLookupErr: gorm.ErrRecordNotFound},
		&fakeProjectReader{project: project},
		&fakeAppAdapter{},
	)
	_, err := service.GetApp(context.Background(), 3, 4, 7)
	assertErrorCode(t, err, errcode.ErrAppNotFound)
}
