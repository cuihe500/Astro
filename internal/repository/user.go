package repository

import (
	"github.com/cuihe500/astro/internal/model"
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
