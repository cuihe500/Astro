package service

import (
	"context"
	"errors"
	"testing"

	"github.com/cuihe500/astro/internal/model"
	"github.com/cuihe500/astro/internal/repository"
	"github.com/cuihe500/astro/pkg/errcode"
	"gorm.io/gorm"
)

type fakeProjectStore struct {
	project        *model.Project
	createdProject *model.Project
	lookupErr      error
	createErr      error
	deleteErr      error
	deleteInvoked  bool
}

func (s *fakeProjectStore) Create(project *model.Project) error {
	s.createdProject = project
	return s.createErr
}

func (s *fakeProjectStore) GetByID(uint) (*model.Project, error) {
	return s.project, s.lookupErr
}

func (s *fakeProjectStore) GetByUserID(uint) ([]model.Project, error) { return nil, nil }

func (s *fakeProjectStore) GetByUserAndName(uint, string) (*model.Project, error) {
	return nil, s.lookupErr
}

func (s *fakeProjectStore) DeleteEmpty(_ uint, _ uint, beforeDelete func(string) error) error {
	s.deleteInvoked = true
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return beforeDelete(s.project.Namespace)
}

func TestCreateProjectCompensatesNamespaceWhenDatabaseFails(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	repo := &fakeProjectStore{lookupErr: gorm.ErrRecordNotFound, createErr: databaseErr}
	adapter := &fakeAppAdapter{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := newProjectService(repo, adapter).CreateProject(ctx, 7, "个人网站")
	assertErrorCode(t, err, errcode.ErrDatabase)
	if adapter.deletedNamespace == "" {
		t.Fatal("数据库写入失败后未删除已创建的命名空间")
	}
	if adapter.deleteNamespaceCtx == nil || adapter.deleteNamespaceCtx.Err() != nil {
		t.Fatal("命名空间补偿不应继承已取消的请求 context")
	}
}

func TestCreateProjectEnsuresStableNamespace(t *testing.T) {
	repo := &fakeProjectStore{lookupErr: gorm.ErrRecordNotFound}
	adapter := &fakeAppAdapter{}
	project, err := newProjectService(repo, adapter).CreateProject(context.Background(), 7, "个人网站")
	if err != nil {
		t.Fatalf("创建项目失败: %v", err)
	}
	if project != repo.createdProject || project.Namespace != adapter.ensuredNamespace {
		t.Fatal("项目记录与已创建命名空间不一致")
	}
	if len(project.Namespace) != len("astro-project-")+36 || project.Namespace[:len("astro-project-")] != "astro-project-" {
		t.Fatalf("项目命名空间格式错误: %q", project.Namespace)
	}
}

func TestDeleteProjectRejectsNonEmptyProject(t *testing.T) {
	repo := &fakeProjectStore{
		project:   &model.Project{BaseModel: model.BaseModel{ID: 3}, UserID: 7, Namespace: "astro-project-test"},
		deleteErr: repository.ErrProjectNotEmpty,
	}
	adapter := &fakeAppAdapter{}
	err := newProjectService(repo, adapter).DeleteProject(context.Background(), 3, 7)
	assertErrorCode(t, err, errcode.ErrProjectNotEmpty)
	if adapter.deletedNamespace != "" {
		t.Fatal("非空项目不应删除命名空间")
	}
}

func TestDeleteProjectReturnsNamespaceFailure(t *testing.T) {
	deleteErr := errors.New("kubernetes unavailable")
	repo := &fakeProjectStore{
		project: &model.Project{BaseModel: model.BaseModel{ID: 3}, UserID: 7, Namespace: "astro-project-test"},
	}
	adapter := &fakeAppAdapter{deleteNamespaceErr: deleteErr}
	err := newProjectService(repo, adapter).DeleteProject(context.Background(), 3, 7)
	assertErrorCode(t, err, errcode.ErrK8sOperation)
	if !repo.deleteInvoked || adapter.deletedNamespace != repo.project.Namespace {
		t.Fatal("删除项目时未使用项目命名空间")
	}
}

func TestGetProjectRejectsOtherUser(t *testing.T) {
	repo := &fakeProjectStore{project: &model.Project{UserID: 9}}
	_, err := newProjectService(repo, &fakeAppAdapter{}).GetProject(3, 7)
	assertErrorCode(t, err, errcode.ErrForbidden)
}

func assertErrorCode(t *testing.T, err error, want errcode.Code) {
	t.Helper()
	var codedError *errcode.Error
	if !errors.As(err, &codedError) || codedError.Code != want {
		t.Fatalf("错误码错误: got %#v, want %d", err, want)
	}
}
