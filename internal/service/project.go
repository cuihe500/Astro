package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/cuihe500/astro/internal/k8s"
	"github.com/cuihe500/astro/internal/model"
	"github.com/cuihe500/astro/internal/repository"
	"github.com/cuihe500/astro/pkg/errcode"
	"github.com/cuihe500/astro/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type projectStore interface {
	Create(project *model.Project) error
	GetByID(id uint) (*model.Project, error)
	GetByUserID(userID uint) ([]model.Project, error)
	GetByUserAndName(userID uint, name string) (*model.Project, error)
	DeleteEmpty(projectID, userID uint, beforeDelete func(string) error) error
}

// ProjectService 项目服务
type ProjectService struct {
	repo    projectStore
	adapter k8s.AppAdapter
}

// NewProjectService 创建项目服务
func NewProjectService() *ProjectService {
	return newProjectService(repository.NewProjectRepository(), k8s.Adapter)
}

func newProjectService(repo projectStore, adapter k8s.AppAdapter) *ProjectService {
	return &ProjectService{repo: repo, adapter: adapter}
}

// CreateProject 创建项目及其命名空间
func (s *ProjectService) CreateProject(ctx context.Context, userID uint, name string) (*model.Project, error) {
	_, err := s.repo.GetByUserAndName(userID, name)
	if err == nil {
		return nil, errcode.New(errcode.ErrProjectExists)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}

	project := &model.Project{
		Name:      name,
		UserID:    userID,
		Namespace: "astro-project-" + uuid.NewString(),
	}
	if err := s.adapter.EnsureNamespace(ctx, project.Namespace); err != nil {
		return nil, errcode.NewWithMsg(errcode.ErrK8sOperation, err.Error())
	}

	if err := s.repo.Create(project); err != nil {
		cleanupErr := s.adapter.DeleteNamespace(ctx, project.Namespace)
		if cleanupErr != nil {
			logger.Error("项目创建失败且命名空间补偿失败",
				zap.Uint("user_id", userID),
				zap.String("namespace", project.Namespace),
				zap.Error(cleanupErr),
			)
			return nil, errcode.NewWithMsg(errcode.ErrProjectCreateFail,
				fmt.Sprintf("数据库错误: %v；清理命名空间失败: %v", err, cleanupErr))
		}
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errcode.New(errcode.ErrProjectExists)
		}
		return nil, errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}

	return project, nil
}

// GetProjects 获取当前用户的项目列表
func (s *ProjectService) GetProjects(userID uint) ([]model.Project, error) {
	projects, err := s.repo.GetByUserID(userID)
	if err != nil {
		return nil, errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}
	return projects, nil
}

// GetProject 获取项目详情
func (s *ProjectService) GetProject(projectID, userID uint) (*model.Project, error) {
	return s.getOwnedProject(projectID, userID)
}

// DeleteProject 删除空项目及其命名空间
func (s *ProjectService) DeleteProject(ctx context.Context, projectID, userID uint) error {
	if _, err := s.getOwnedProject(projectID, userID); err != nil {
		return err
	}

	err := s.repo.DeleteEmpty(projectID, userID, func(namespace string) error {
		if deleteErr := s.adapter.DeleteNamespace(ctx, namespace); deleteErr != nil {
			return errcode.NewWithMsg(errcode.ErrK8sOperation, deleteErr.Error())
		}
		return nil
	})
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrProjectNotEmpty) {
		return errcode.New(errcode.ErrProjectNotEmpty)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errcode.New(errcode.ErrProjectNotFound)
	}
	var codedError *errcode.Error
	if errors.As(err, &codedError) {
		return codedError
	}
	return errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
}

func (s *ProjectService) getOwnedProject(projectID, userID uint) (*model.Project, error) {
	project, err := s.repo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.New(errcode.ErrProjectNotFound)
		}
		return nil, errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}
	if project.UserID != userID {
		return nil, errcode.New(errcode.ErrForbidden)
	}
	return project, nil
}
