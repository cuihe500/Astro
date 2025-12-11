package service

import (
	"errors"
	"time"

	"github.com/cuihe500/astro/internal/model"
	"github.com/cuihe500/astro/internal/repository"
	"github.com/cuihe500/astro/pkg/config"
	"github.com/cuihe500/astro/pkg/errcode"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService() *UserService {
	return &UserService{
		repo: repository.NewUserRepository(),
	}
}

// Register 用户注册
func (s *UserService) Register(username, password, email string) error {
	// 检查用户是否已存在
	_, err := s.repo.GetUserByUsername(username)
	if err == nil {
		return errcode.New(errcode.ErrUserExists)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errcode.NewWithMsg(errcode.ErrInternal, err.Error())
	}

	// 创建用户
	user := &model.User{
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
	}
	if err := s.repo.CreateUser(user); err != nil {
		return errcode.NewWithMsg(errcode.ErrRegisterFailed, err.Error())
	}
	return nil
}

// Login 用户登录，返回 token 和用户信息
func (s *UserService) Login(username, password string) (string, *model.User, error) {
	// 查询用户
	user, err := s.repo.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, errcode.New(errcode.ErrLoginFailed)
		}
		return "", nil, errcode.NewWithMsg(errcode.ErrDatabase, err.Error())
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, errcode.New(errcode.ErrLoginFailed)
	}

	// 生成 JWT
	token, err := s.generateToken(user.ID, user.UUID)
	if err != nil {
		return "", nil, errcode.NewWithMsg(errcode.ErrInternal, err.Error())
	}

	return token, user, nil
}

// generateToken 生成 JWT token
func (s *UserService) generateToken(userID uint, uuid string) (string, error) {
	cfg := config.GlobalConfig.JWT

	// 解析过期时间
	expire, err := time.ParseDuration(cfg.Expire)
	if err != nil {
		expire = 24 * time.Hour
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"uuid":    uuid,
		"exp":     time.Now().Add(expire).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.Secret))
}
