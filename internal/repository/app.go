package repository

import (
	"github.com/cuihe500/astro/internal/model"
)

// AppRepository 应用数据仓库
type AppRepository struct{}

// NewAppRepository 创建应用仓库
func NewAppRepository() *AppRepository {
	return &AppRepository{}
}

// Create 创建应用记录
func (r *AppRepository) Create(app *model.App) error {
	return DB.Create(app).Error
}

// Update 更新应用信息
func (r *AppRepository) Update(app *model.App) error {
	return DB.Save(app).Error
}

// Delete 删除应用记录（软删除）
func (r *AppRepository) Delete(id uint) error {
	return DB.Delete(&model.App{}, id).Error
}

// GetByID 按 ID 查询应用
func (r *AppRepository) GetByID(id uint) (*model.App, error) {
	var app model.App
	if err := DB.First(&app, id).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// GetByUserID 按用户 ID 查询应用列表
func (r *AppRepository) GetByUserID(userID uint) ([]model.App, error) {
	var apps []model.App
	if err := DB.Where("user_id = ?", userID).Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}

// GetByUserAndName 按用户 ID 和应用名查询
func (r *AppRepository) GetByUserAndName(userID uint, name string) (*model.App, error) {
	var app model.App
	if err := DB.Where("user_id = ? AND name = ?", userID, name).First(&app).Error; err != nil {
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
