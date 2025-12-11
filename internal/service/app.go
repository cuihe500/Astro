package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/cuihe500/astro/internal/k8s"
	"github.com/cuihe500/astro/internal/model"
	"github.com/cuihe500/astro/internal/repository"
	"github.com/cuihe500/astro/pkg/errcode"
	"gorm.io/gorm"
)

// AppService 应用服务
type AppService struct {
	repo    *repository.AppRepository
	adapter k8s.AppAdapter
}

// NewAppService 创建应用服务
func NewAppService() *AppService {
	return &AppService{
		repo:    repository.NewAppRepository(),
		adapter: k8s.Adapter,
	}
}

// CreateAppRequest 创建应用请求
type CreateAppRequest struct {
	Name     string
	Image    string
	Replicas int
	Port     int
	UserID   uint
}

// CreateApp 创建应用
func (s *AppService) CreateApp(ctx context.Context, req CreateAppRequest) (*model.App, error) {
	// 检查应用名是否重复
	_, err := s.repo.GetByUserAndName(req.UserID, req.Name)
	if err == nil {
		return nil, errcode.New(errcode.ErrAppExists)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}

	// 构建命名空间
	namespace := fmt.Sprintf("astro-user-%d", req.UserID)

	// 创建数据库记录
	app := &model.App{
		Name:      req.Name,
		Image:     req.Image,
		Replicas:  req.Replicas,
		Status:    "pending",
		UserID:    req.UserID,
		Namespace: namespace,
	}
	if err := s.repo.Create(app); err != nil {
		return nil, errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}

	// 调用 K8s Adapter 创建应用
	spec := k8s.AppSpec{
		Name:      req.Name,
		Namespace: namespace,
		Image:     req.Image,
		Replicas:  int32(req.Replicas),
		Port:      int32(req.Port),
	}
	if err := s.adapter.CreateApp(ctx, spec); err != nil {
		// 创建 K8s 资源失败，删除数据库记录
		_ = s.repo.Delete(app.ID)
		return nil, errcode.NewWithMsg(errcode.ErrAppCreateFailed, err.Error())
	}

	// 异步同步状态
	go s.syncAppStatus(context.Background(), app.ID, app.Name, namespace)

	return app, nil
}

// DeleteApp 删除应用
func (s *AppService) DeleteApp(ctx context.Context, appID, userID uint) error {
	app, err := s.repo.GetByID(appID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.New(errcode.ErrAppNotFound)
		}
		return errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}

	// 权限检查
	if app.UserID != userID {
		return errcode.New(errcode.ErrForbidden)
	}

	// 删除 K8s 资源
	if err := s.adapter.DeleteApp(ctx, app.Name, app.Namespace); err != nil {
		return errcode.NewWithMsg(errcode.ErrK8sOperation, err.Error())
	}

	// 删除数据库记录
	if err := s.repo.Delete(appID); err != nil {
		return errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}

	return nil
}

// StartApp 启动应用
func (s *AppService) StartApp(ctx context.Context, appID, userID uint) error {
	app, err := s.getAppWithPermission(appID, userID)
	if err != nil {
		return err
	}

	// 恢复到原来的副本数（至少为1）
	replicas := app.Replicas
	if replicas == 0 {
		replicas = 1
	}

	if err := s.adapter.ScaleApp(ctx, app.Name, app.Namespace, int32(replicas)); err != nil {
		return errcode.NewWithMsg(errcode.ErrK8sOperation, err.Error())
	}

	_ = s.repo.UpdateStatus(appID, "starting")
	go s.syncAppStatus(context.Background(), appID, app.Name, app.Namespace)

	return nil
}

// StopApp 停止应用
func (s *AppService) StopApp(ctx context.Context, appID, userID uint) error {
	app, err := s.getAppWithPermission(appID, userID)
	if err != nil {
		return err
	}

	if err := s.adapter.ScaleApp(ctx, app.Name, app.Namespace, 0); err != nil {
		return errcode.NewWithMsg(errcode.ErrK8sOperation, err.Error())
	}

	_ = s.repo.UpdateStatus(appID, "stopped")
	_ = s.repo.UpdateReplicas(appID, 0)

	return nil
}

// RestartApp 重启应用
func (s *AppService) RestartApp(ctx context.Context, appID, userID uint) error {
	app, err := s.getAppWithPermission(appID, userID)
	if err != nil {
		return err
	}

	if err := s.adapter.RestartApp(ctx, app.Name, app.Namespace); err != nil {
		return errcode.NewWithMsg(errcode.ErrK8sOperation, err.Error())
	}

	_ = s.repo.UpdateStatus(appID, "restarting")
	go s.syncAppStatus(context.Background(), appID, app.Name, app.Namespace)

	return nil
}

// GetApps 获取用户的应用列表
func (s *AppService) GetApps(ctx context.Context, userID uint) ([]model.App, error) {
	apps, err := s.repo.GetByUserID(userID)
	if err != nil {
		return nil, errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}

	// 异步同步所有应用状态
	for _, app := range apps {
		go s.syncAppStatus(context.Background(), app.ID, app.Name, app.Namespace)
	}

	return apps, nil
}

// GetApp 获取应用详情
func (s *AppService) GetApp(ctx context.Context, appID, userID uint) (*model.App, error) {
	app, err := s.getAppWithPermission(appID, userID)
	if err != nil {
		return nil, err
	}

	// 同步状态后重新查询
	s.syncAppStatus(ctx, appID, app.Name, app.Namespace)
	return s.repo.GetByID(appID)
}

// GetAppLogs 获取应用日志
func (s *AppService) GetAppLogs(ctx context.Context, appID, userID uint, lines int64) (string, error) {
	app, err := s.getAppWithPermission(appID, userID)
	if err != nil {
		return "", err
	}

	logs, err := s.adapter.GetAppLogs(ctx, app.Name, app.Namespace, lines)
	if err != nil {
		return "", errcode.NewWithMsg(errcode.ErrK8sOperation, err.Error())
	}

	return logs, nil
}

// getAppWithPermission 获取应用并检查权限
func (s *AppService) getAppWithPermission(appID, userID uint) (*model.App, error) {
	app, err := s.repo.GetByID(appID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.New(errcode.ErrAppNotFound)
		}
		return nil, errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}

	if app.UserID != userID {
		return nil, errcode.New(errcode.ErrForbidden)
	}

	return app, nil
}

// syncAppStatus 同步应用状态
func (s *AppService) syncAppStatus(ctx context.Context, appID uint, name, namespace string) {
	status, err := s.adapter.GetAppStatus(ctx, name, namespace)
	if err != nil {
		return
	}

	_ = s.repo.UpdateStatus(appID, status.Status)
	if status.Replicas > 0 {
		_ = s.repo.UpdateReplicas(appID, int(status.Replicas))
	}
}
