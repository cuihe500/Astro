package handler

import (
	"net/http"

	"github.com/cuihe500/astro/pkg/errcode"
	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    errcode.Success.Int(),
		Message: errcode.Success.Message(),
		Data:    data,
	})
}

// Error 错误响应（使用错误码枚举）
func Error(c *gin.Context, code errcode.Code, message string) {
	msg := message
	if msg == "" {
		msg = code.Message()
	}
	c.JSON(http.StatusOK, Response{
		Code:    code.Int(),
		Message: msg,
	})
}

// ErrorWithCode 使用错误码的默认消息
func ErrorWithCode(c *gin.Context, code errcode.Code) {
	Error(c, code, code.Message())
}

// BadRequest 参数错误响应
func BadRequest(c *gin.Context, message string) {
	Error(c, errcode.ErrBadRequest, message)
}

// Unauthorized 未授权响应
func Unauthorized(c *gin.Context, message string) {
	Error(c, errcode.ErrUnauthorized, message)
}

// Forbidden 禁止访问响应
func Forbidden(c *gin.Context, message string) {
	Error(c, errcode.ErrForbidden, message)
}

// NotFound 资源不存在响应
func NotFound(c *gin.Context, message string) {
	Error(c, errcode.ErrNotFound, message)
}

// InternalError 服务器内部错误响应
func InternalError(c *gin.Context, message string) {
	Error(c, errcode.ErrInternal, message)
}

// HandleError 处理 service 层返回的错误
func HandleError(c *gin.Context, err error) {
	e := errcode.FromError(err)
	Error(c, e.Code, e.Msg)
}
