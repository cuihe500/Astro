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

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	// 自动迁移
	if err := db.AutoMigrate(&model.User{}, &model.App{}); err != nil {
		return err
	}

	DB = db
	return nil
}
