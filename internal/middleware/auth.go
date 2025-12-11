package middleware

import (
	"errors"
	"strings"

	"github.com/cuihe500/astro/internal/handler"
	"github.com/cuihe500/astro/pkg/config"
	"github.com/cuihe500/astro/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const contextKeyUserID = "user_id"

// Auth JWT 认证中间件
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			handler.ErrorWithCode(c, errcode.ErrUnauthorized)
			c.Abort()
			return
		}

		// 检查 Bearer 前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			handler.ErrorWithCode(c, errcode.ErrTokenInvalid)
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 解析并验证 token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(config.GlobalConfig.JWT.Secret), nil
		})

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				handler.ErrorWithCode(c, errcode.ErrTokenExpired)
			} else {
				handler.ErrorWithCode(c, errcode.ErrTokenInvalid)
			}
			c.Abort()
			return
		}

		// 提取 claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			handler.ErrorWithCode(c, errcode.ErrTokenInvalid)
			c.Abort()
			return
		}

		// 提取 user_id
		userID, ok := claims["user_id"].(float64)
		if !ok {
			handler.ErrorWithCode(c, errcode.ErrTokenInvalid)
			c.Abort()
			return
		}

		c.Set(contextKeyUserID, uint(userID))
		c.Next()
	}
}

// GetUserID 从 Context 中获取当前登录用户 ID
func GetUserID(c *gin.Context) (uint, bool) {
	userID, exists := c.Get(contextKeyUserID)
	if !exists {
		return 0, false
	}
	id, ok := userID.(uint)
	return id, ok
}
