package repository

import (
	"github.com/cuihe500/astro/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AppRepository 应用数据仓库
type AppRepository struct{}

// NewAppRepository 创建应用仓库
func NewAppRepository() *AppRepository {
	return &AppRepository{}
}

// CreateInProject 锁定所属项目，并在提交前完成应用资源创建。
func (r *AppRepository) CreateInProject(app *model.App, userID uint, beforeCommit func(string) error) (*model.Project, error) {
	var project model.Project
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", app.ProjectID, userID).
			First(&project).Error; err != nil {
			return err
		}
		if err := tx.Create(app).Error; err != nil {
			return err
		}
		return beforeCommit(project.Namespace)
	})
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// Update 更新应用信息
func (r *AppRepository) Update(app *model.App) error {
	return DB.Save(app).Error
}

// Delete 硬删除应用记录
func (r *AppRepository) Delete(id uint) error {
	return DB.Unscoped().Delete(&model.App{}, id).Error
}

// GetByProjectAndID 按项目和 ID 查询应用
func (r *AppRepository) GetByProjectAndID(projectID, id uint) (*model.App, error) {
	var app model.App
	if err := DB.Where("project_id = ? AND id = ?", projectID, id).First(&app).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// GetByProjectID 按项目 ID 查询应用列表
func (r *AppRepository) GetByProjectID(projectID uint) ([]model.App, error) {
	var apps []model.App
	if err := DB.Where("project_id = ?", projectID).Order("created_at DESC").Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}

// GetByProjectAndName 按项目 ID 和应用名查询
func (r *AppRepository) GetByProjectAndName(projectID uint, name string) (*model.App, error) {
	var app model.App
	if err := DB.Where("project_id = ? AND name = ?", projectID, name).First(&app).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// UpdateStatus 更新应用状态
func (r *AppRepository) UpdateStatus(id uint, status string) error {
	return DB.Model(&model.App{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateReplicas 更新应用副本数
func (r *AppRepository) UpdateReplicas(id uint, replicas int) error {
	return DB.Model(&model.App{}).Where("id = ?", id).Update("replicas", replicas).Error
}
