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
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type appStore interface {
	CreateInProject(app *model.App, userID uint, beforeCommit func(string) error) (*model.Project, error)
	Delete(id uint) error
	GetByProjectAndID(projectID, id uint) (*model.App, error)
	GetByProjectID(projectID uint) ([]model.App, error)
	GetByProjectAndName(projectID uint, name string) (*model.App, error)
	UpdateStatus(id uint, status string) error
	UpdateReplicas(id uint, replicas int) error
}

type projectReader interface {
	GetByID(id uint) (*model.Project, error)
}

// AppService 应用服务
type AppService struct {
	repo     appStore
	projects projectReader
	adapter  k8s.AppAdapter
}

// NewAppService 创建应用服务
func NewAppService() *AppService {
	return newAppService(repository.NewAppRepository(), repository.NewProjectRepository(), k8s.Adapter)
}

func newAppService(repo appStore, projects projectReader, adapter k8s.AppAdapter) *AppService {
	return &AppService{repo: repo, projects: projects, adapter: adapter}
}

// CreateAppRequest 创建应用请求
type CreateAppRequest struct {
	Name      string
	Image     string
	Replicas  int
	Config    model.AppConfig
	ProjectID uint
	UserID    uint
}

// CreateApp 创建应用
func (s *AppService) CreateApp(ctx context.Context, req CreateAppRequest) (*model.App, error) {
	project, err := s.getOwnedProject(req.ProjectID, req.UserID)
	if err != nil {
		return nil, err
	}

	_, err = s.repo.GetByProjectAndName(req.ProjectID, req.Name)
	if err == nil {
		return nil, errcode.New(errcode.ErrAppExists)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}

	app := &model.App{
		Name:      req.Name,
		Image:     req.Image,
		Replicas:  req.Replicas,
		Status:    "pending",
		Config:    req.Config,
		ProjectID: req.ProjectID,
	}
	if err := s.adapter.ValidateAppReferences(ctx, project.Namespace, req.Config); err != nil {
		return nil, errcode.NewWithMsg(errcode.ErrAppCreateFailed, "引用资源校验失败: "+err.Error())
	}
	var createdNamespace string
	resourcesTouched := false
	project, err = s.repo.CreateInProject(app, req.UserID, func(namespace string) error {
		createdNamespace = namespace
		spec := k8s.AppSpec{
			Name:      req.Name,
			Namespace: namespace,
			Image:     req.Image,
			Replicas:  int32(req.Replicas),
			Config:    req.Config,
		}
		resourcesTouched = true
		if createErr := s.adapter.CreateApp(ctx, spec); createErr != nil {
			return errcode.NewWithMsg(errcode.ErrAppCreateFailed, createErr.Error())
		}
		return nil
	})
	if err != nil {
		if resourcesTouched {
			if cleanupErr := s.adapter.DeleteApp(context.WithoutCancel(ctx), app.Name, createdNamespace); cleanupErr != nil {
				logger.Error("应用事务失败且 Kubernetes 资源补偿失败",
					zap.Uint("project_id", req.ProjectID),
					zap.String("app_name", app.Name),
					zap.String("namespace", createdNamespace),
					zap.Error(cleanupErr),
				)
				return nil, errcode.NewWithMsg(errcode.ErrAppCreateFailed,
					fmt.Sprintf("应用创建错误: %v；清理 Kubernetes 资源失败: %v", err, cleanupErr))
			}
		}
		var codedError *errcode.Error
		if errors.As(err, &codedError) {
			return nil, codedError
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.New(errcode.ErrProjectNotFound)
		}
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errcode.New(errcode.ErrAppExists)
		}
		return nil, errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}

	go s.syncAppStatus(context.Background(), app.ID, app.Name, project.Namespace)
	return app, nil
}

// DeleteApp 删除应用
func (s *AppService) DeleteApp(ctx context.Context, projectID, appID, userID uint) error {
	app, project, err := s.getAppWithPermission(projectID, appID, userID)
	if err != nil {
		return err
	}
	if err := s.adapter.DeleteApp(ctx, app.Name, project.Namespace); err != nil {
		return errcode.NewWithMsg(errcode.ErrK8sOperation, err.Error())
	}
	if err := s.repo.Delete(appID); err != nil {
		return errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}
	return nil
}

// StartApp 启动应用
func (s *AppService) StartApp(ctx context.Context, projectID, appID, userID uint) error {
	app, project, err := s.getAppWithPermission(projectID, appID, userID)
	if err != nil {
		return err
	}
	replicas := app.Replicas
	if replicas == 0 {
		replicas = 1
	}
	if err := s.adapter.ScaleApp(ctx, app.Name, project.Namespace, int32(replicas)); err != nil {
		return errcode.NewWithMsg(errcode.ErrK8sOperation, err.Error())
	}
	if err := s.repo.UpdateStatus(appID, "starting"); err != nil {
		return errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}
	go s.syncAppStatus(context.Background(), appID, app.Name, project.Namespace)
	return nil
}

// StopApp 停止应用
func (s *AppService) StopApp(ctx context.Context, projectID, appID, userID uint) error {
	app, project, err := s.getAppWithPermission(projectID, appID, userID)
	if err != nil {
		return err
	}
	if err := s.adapter.ScaleApp(ctx, app.Name, project.Namespace, 0); err != nil {
		return errcode.NewWithMsg(errcode.ErrK8sOperation, err.Error())
	}
	if err := s.repo.UpdateStatus(appID, "stopped"); err != nil {
		return errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}
	if err := s.repo.UpdateReplicas(appID, 0); err != nil {
		return errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}
	return nil
}

// RestartApp 重启应用
func (s *AppService) RestartApp(ctx context.Context, projectID, appID, userID uint) error {
	app, project, err := s.getAppWithPermission(projectID, appID, userID)
	if err != nil {
		return err
	}
	if err := s.adapter.RestartApp(ctx, app.Name, project.Namespace); err != nil {
		return errcode.NewWithMsg(errcode.ErrK8sOperation, err.Error())
	}
	if err := s.repo.UpdateStatus(appID, "restarting"); err != nil {
		return errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}
	go s.syncAppStatus(context.Background(), appID, app.Name, project.Namespace)
	return nil
}

// GetApps 获取项目的应用列表
func (s *AppService) GetApps(_ context.Context, projectID, userID uint) ([]model.App, error) {
	project, err := s.getOwnedProject(projectID, userID)
	if err != nil {
		return nil, err
	}
	apps, err := s.repo.GetByProjectID(projectID)
	if err != nil {
		return nil, errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}
	for _, app := range apps {
		go s.syncAppStatus(context.Background(), app.ID, app.Name, project.Namespace)
	}
	return apps, nil
}

// GetApp 获取应用详情
func (s *AppService) GetApp(ctx context.Context, projectID, appID, userID uint) (*model.App, error) {
	app, project, err := s.getAppWithPermission(projectID, appID, userID)
	if err != nil {
		return nil, err
	}
	s.syncAppStatus(ctx, appID, app.Name, project.Namespace)
	app, err = s.repo.GetByProjectAndID(projectID, appID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.New(errcode.ErrAppNotFound)
		}
		return nil, errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}
	return app, nil
}

// GetAppLogs 获取应用日志
func (s *AppService) GetAppLogs(ctx context.Context, projectID, appID, userID uint, lines int64) (string, error) {
	app, project, err := s.getAppWithPermission(projectID, appID, userID)
	if err != nil {
		return "", err
	}
	logs, err := s.adapter.GetAppLogs(ctx, app.Name, project.Namespace, lines)
	if err != nil {
		return "", errcode.NewWithMsg(errcode.ErrK8sOperation, err.Error())
	}
	return logs, nil
}

func (s *AppService) getOwnedProject(projectID, userID uint) (*model.Project, error) {
	project, err := s.projects.GetByID(projectID)
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

func (s *AppService) getAppWithPermission(projectID, appID, userID uint) (*model.App, *model.Project, error) {
	project, err := s.getOwnedProject(projectID, userID)
	if err != nil {
		return nil, nil, err
	}
	app, err := s.repo.GetByProjectAndID(projectID, appID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errcode.New(errcode.ErrAppNotFound)
		}
		return nil, nil, errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}
	return app, project, nil
}

func (s *AppService) syncAppStatus(ctx context.Context, appID uint, name, namespace string) {
	status, err := s.adapter.GetAppStatus(ctx, name, namespace)
	if err != nil {
		logger.Warn("同步应用状态失败", zap.Uint("app_id", appID), zap.Error(err))
		return
	}
	if err := s.repo.UpdateStatus(appID, status.Status); err != nil {
		logger.Error("写入应用状态失败", zap.Uint("app_id", appID), zap.Error(err))
		return
	}
	if status.Replicas > 0 {
		if err := s.repo.UpdateReplicas(appID, int(status.Replicas)); err != nil {
			logger.Error("写入应用副本数失败", zap.Uint("app_id", appID), zap.Error(err))
		}
	}
}
