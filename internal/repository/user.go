package repository

import (
	"github.com/cuihe500/astro/internal/model"
	"gorm.io/gorm"
)

type UserRepository struct{}

func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// CreateUser 创建用户
func (r *UserRepository) CreateUser(user *model.User) error {
	return DB.Create(user).Error
}

// GetUserByUsername 通过用户名查询用户
func (r *UserRepository) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	if err := DB.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail 通过邮箱查询用户
func (r *UserRepository) GetUserByEmail(email string) (*model.User, error) {
	var user model.User
	if err := DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID 通过 ID 查询用户
func (r *UserRepository) GetUserByID(id uint) (*model.User, error) {
	var user model.User
	if err := DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUUID 通过 UUID 查询用户
func (r *UserRepository) GetUserByUUID(uuid string) (*model.User, error) {
	var user model.User
	if err := DB.Where("uuid = ?", uuid).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetOAuthIdentity 查询 OAuth2 身份
func (r *UserRepository) GetOAuthIdentity(provider, providerUserID string) (*model.OAuthIdentity, error) {
	var identity model.OAuthIdentity
	if err := DB.Where("provider = ? AND provider_user_id = ?", provider, providerUserID).First(&identity).Error; err != nil {
		return nil, err
	}
	return &identity, nil
}

// CreateOAuthUser 创建 OAuth2 用户和身份
func (r *UserRepository) CreateOAuthUser(user *model.User, identity *model.OAuthIdentity) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		identity.UserID = user.ID
		return tx.Create(identity).Error
	})
}
