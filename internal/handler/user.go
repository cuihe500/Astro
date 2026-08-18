package handler

import (
	"github.com/cuihe500/astro/internal/service"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	svc       *service.UserService
	oauth2Svc *service.OAuth2Service
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		svc:       service.NewUserService(),
		oauth2Svc: service.NewOAuth2Service(),
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required" example:"johndoe"`
	Password string `json:"password" binding:"required" example:"password123"`
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required" example:"johndoe"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	UUID  string `json:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// OAuth2LoginResponse OAuth2 登录入口响应
type OAuth2LoginResponse struct {
	AuthURL string `json:"auth_url" example:"https://auth.example.com/application/o/authorize/?client_id=astro"`
}

// Register 用户注册
// @Summary 用户注册
// @Description 创建新用户账号
// @Tags 用户
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "注册信息"
// @Success 200 {object} Response "注册成功"
// @Failure 400 {object} Response "参数错误"
// @Failure 500 {object} Response "服务器错误"
// @Router /register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	if err := h.svc.Register(req.Username, req.Password, req.Email); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, nil)
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录获取 Token
// @Tags 用户
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录信息"
// @Success 200 {object} Response{data=LoginResponse} "登录成功"
// @Failure 400 {object} Response "参数错误"
// @Failure 401 {object} Response "认证失败"
// @Router /login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	token, user, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, LoginResponse{Token: token, UUID: user.UUID})
}

// OAuth2Login 获取 OAuth2 授权 URL
// @Summary OAuth2 登录入口
// @Description 获取 OAuth2/OIDC Provider 授权 URL
// @Tags 用户
// @Produce json
// @Param provider path string true "Provider 名称" example(authentik)
// @Success 200 {object} Response{data=OAuth2LoginResponse} "获取成功"
// @Failure 400 {object} Response "Provider 不可用"
// @Router /oauth2/{provider}/login [get]
func (h *UserHandler) OAuth2Login(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		BadRequest(c, "Provider 不能为空")
		return
	}

	authURL, err := h.oauth2Svc.BuildAuthURL(provider)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, OAuth2LoginResponse{AuthURL: authURL})
}

// OAuth2Callback 处理 OAuth2 回调
// @Summary OAuth2 回调
// @Description 使用授权码换取本系统 JWT
// @Tags 用户
// @Produce json
// @Param provider path string true "Provider 名称" example(authentik)
// @Param code query string true "授权码"
// @Param state query string true "状态参数"
// @Success 200 {object} Response{data=LoginResponse} "登录成功"
// @Failure 400 {object} Response "参数错误"
// @Failure 401 {object} Response "认证失败"
// @Router /oauth2/{provider}/callback [get]
func (h *UserHandler) OAuth2Callback(c *gin.Context) {
	provider := c.Param("provider")
	code := c.Query("code")
	state := c.Query("state")
	if provider == "" || code == "" || state == "" {
		BadRequest(c, "provider、code、state 不能为空")
		return
	}

	token, user, err := h.oauth2Svc.Callback(c.Request.Context(), provider, code, state)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, LoginResponse{Token: token, UUID: user.UUID})
}

// RegisterRoutes 注册用户相关路由
func RegisterUserRoutes(r *gin.RouterGroup) {
	h := NewUserHandler()
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.GET("/oauth2/:provider/login", h.OAuth2Login)
	r.GET("/oauth2/:provider/callback", h.OAuth2Callback)
}
