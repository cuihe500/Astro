package repository

import (
	"fmt"

	"github.com/cuihe500/astro/internal/model"
	"github.com/cuihe500/astro/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init 初始化数据库连接
func Init(cfg *config.DatabaseConfig) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.Charset)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		return err
	}

	if err := prepareAppSchema(db); err != nil {
		return err
	}

	// 自动迁移
	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.App{}, &model.OAuthIdentity{}); err != nil {
		return err
	}

	DB = db
	return nil
}

func prepareAppSchema(db *gorm.DB) error {
	migrator := db.Migrator()
	appModel := &model.App{}
	legacySchema := migrator.HasTable(appModel) &&
		migrator.HasColumn(appModel, "user_id") &&
		migrator.HasColumn(appModel, "namespace") &&
		!migrator.HasColumn(appModel, "project_id")
	if !legacySchema {
		return nil
	}

	var appCount int64
	if err := db.Table("apps").Where("deleted_at IS NULL").Count(&appCount).Error; err != nil {
		return fmt.Errorf("检查旧应用数据失败: %w", err)
	}
	if appCount > 0 {
		return fmt.Errorf("检测到 %d 个旧应用，请先按清单删除其 Kubernetes 资源和数据库记录", appCount)
	}

	if err := migrator.DropTable(appModel); err != nil {
		return fmt.Errorf("重建应用表前删除旧表失败: %w", err)
	}
	return nil
}
