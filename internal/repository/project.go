package repository

import (
	"errors"

	"github.com/cuihe500/astro/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrProjectNotEmpty 表示项目仍包含应用。
var ErrProjectNotEmpty = errors.New("项目仍包含应用")

// ProjectRepository 项目数据仓库
type ProjectRepository struct{}

// NewProjectRepository 创建项目仓库
func NewProjectRepository() *ProjectRepository {
	return &ProjectRepository{}
}

// Create 创建项目记录
func (r *ProjectRepository) Create(project *model.Project) error {
	return DB.Create(project).Error
}

// GetByID 按 ID 查询项目
func (r *ProjectRepository) GetByID(id uint) (*model.Project, error) {
	var project model.Project
	if err := DB.First(&project, id).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

// GetByUserID 按用户 ID 查询项目列表
func (r *ProjectRepository) GetByUserID(userID uint) ([]model.Project, error) {
	var projects []model.Project
	if err := DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

// GetByUserAndName 按用户 ID 和项目名查询项目
func (r *ProjectRepository) GetByUserAndName(userID uint, name string) (*model.Project, error) {
	var project model.Project
	if err := DB.Where("user_id = ? AND name = ?", userID, name).First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

// DeleteEmpty 锁定项目并在删除前确认项目为空。
func (r *ProjectRepository) DeleteEmpty(projectID, userID uint, beforeDelete func(string) error) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var project model.Project
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", projectID, userID).
			First(&project).Error; err != nil {
			return err
		}

		var appCount int64
		if err := tx.Model(&model.App{}).Where("project_id = ?", project.ID).Count(&appCount).Error; err != nil {
			return err
		}
		if appCount > 0 {
			return ErrProjectNotEmpty
		}

		if err := beforeDelete(project.Namespace); err != nil {
			return err
		}
		return tx.Unscoped().Delete(&project).Error
	})
}
