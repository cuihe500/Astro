package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel 基础模型
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// User 用户模型
type User struct {
	BaseModel
	UUID     string `gorm:"type:char(36);uniqueIndex;not null" json:"uuid"`
	Username string `gorm:"size:64;uniqueIndex;not null" json:"username"`
	Password string `gorm:"size:128;not null" json:"-"`
	Email    string `gorm:"size:128;uniqueIndex" json:"email"`
	Status   int    `gorm:"default:1" json:"status"`
}

// BeforeCreate 创建用户前自动生成 UUID
func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.UUID = uuid.New().String()
	return nil
}

// OAuthIdentity OAuth2 身份模型
type OAuthIdentity struct {
	BaseModel
	Provider       string `gorm:"size:64;uniqueIndex:idx_oauth_provider_user;not null" json:"provider"`
	ProviderUserID string `gorm:"size:128;uniqueIndex:idx_oauth_provider_user;not null" json:"provider_user_id"`
	UserID         uint   `gorm:"index;not null" json:"user_id"`
}

// App 应用模型
type App struct {
	BaseModel
	Name      string `gorm:"size:64;not null" json:"name"`
	Image     string `gorm:"size:256;not null" json:"image"`
	Replicas  int    `gorm:"default:1" json:"replicas"`
	Status    string `gorm:"size:32;default:stopped" json:"status"`
	UserID    uint   `gorm:"index;not null" json:"user_id"`
	Namespace string `gorm:"size:64" json:"namespace"`
}
