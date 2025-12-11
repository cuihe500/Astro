package handler

import (
	"context"
	"strconv"

	"github.com/cuihe500/astro/internal/service"
	"github.com/gin-gonic/gin"
)

// AppHandler 应用处理器
type AppHandler struct {
	svc *service.AppService
}

// NewAppHandler 创建应用处理器
func NewAppHandler() *AppHandler {
	return &AppHandler{
		svc: service.NewAppService(),
	}
}

// CreateAppRequest 创建应用请求
type CreateAppRequest struct {
	Name     string `json:"name" binding:"required" example:"my-nginx"`
	Image    string `json:"image" binding:"required" example:"nginx:latest"`
	Replicas int    `json:"replicas" binding:"required,min=0,max=10" example:"2"`
	Port     int    `json:"port" example:"80"`
}

// AppLogsResponse 日志响应
type AppLogsResponse struct {
	Logs string `json:"logs"`
}

// CreateApp 创建应用
// @Summary 创建应用
// @Description 创建一个新的容器应用
// @Tags 应用
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body CreateAppRequest true "应用信息"
// @Success 200 {object} Response "创建成功"
// @Failure 400 {object} Response "参数错误"
// @Failure 401 {object} Response "未授权"
// @Router /apps [post]
func (h *AppHandler) CreateApp(c *gin.Context) {
	var req CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		Unauthorized(c, "未登录")
		return
	}

	app, err := h.svc.CreateApp(context.Background(), service.CreateAppRequest{
		Name:     req.Name,
		Image:    req.Image,
		Replicas: req.Replicas,
		Port:     req.Port,
		UserID:   userID,
	})
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, app)
}

// GetApps 获取应用列表
// @Summary 获取应用列表
// @Description 获取当前用户的所有应用
// @Tags 应用
// @Produce json
// @Security Bearer
// @Success 200 {object} Response "成功"
// @Failure 401 {object} Response "未授权"
// @Router /apps [get]
func (h *AppHandler) GetApps(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		Unauthorized(c, "未登录")
		return
	}

	apps, err := h.svc.GetApps(context.Background(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, apps)
}

// GetApp 获取应用详情
// @Summary 获取应用详情
// @Description 获取指定应用的详细信息
// @Tags 应用
// @Produce json
// @Security Bearer
// @Param id path int true "应用ID"
// @Success 200 {object} Response "成功"
// @Failure 401 {object} Response "未授权"
// @Failure 404 {object} Response "应用不存在"
// @Router /apps/{id} [get]
func (h *AppHandler) GetApp(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		BadRequest(c, "无效的应用ID")
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		Unauthorized(c, "未登录")
		return
	}

	app, err := h.svc.GetApp(context.Background(), uint(appID), userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, app)
}

// DeleteApp 删除应用
// @Summary 删除应用
// @Description 删除指定的应用
// @Tags 应用
// @Produce json
// @Security Bearer
// @Param id path int true "应用ID"
// @Success 200 {object} Response "删除成功"
// @Failure 401 {object} Response "未授权"
// @Failure 404 {object} Response "应用不存在"
// @Router /apps/{id} [delete]
func (h *AppHandler) DeleteApp(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		BadRequest(c, "无效的应用ID")
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		Unauthorized(c, "未登录")
		return
	}

	if err := h.svc.DeleteApp(context.Background(), uint(appID), userID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, nil)
}

// StartApp 启动应用
// @Summary 启动应用
// @Description 启动指定的应用
// @Tags 应用
// @Produce json
// @Security Bearer
// @Param id path int true "应用ID"
// @Success 200 {object} Response "启动成功"
// @Failure 401 {object} Response "未授权"
// @Failure 404 {object} Response "应用不存在"
// @Router /apps/{id}/start [post]
func (h *AppHandler) StartApp(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		BadRequest(c, "无效的应用ID")
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		Unauthorized(c, "未登录")
		return
	}

	if err := h.svc.StartApp(context.Background(), uint(appID), userID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, nil)
}

// StopApp 停止应用
// @Summary 停止应用
// @Description 停止指定的应用
// @Tags 应用
// @Produce json
// @Security Bearer
// @Param id path int true "应用ID"
// @Success 200 {object} Response "停止成功"
// @Failure 401 {object} Response "未授权"
// @Failure 404 {object} Response "应用不存在"
// @Router /apps/{id}/stop [post]
func (h *AppHandler) StopApp(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		BadRequest(c, "无效的应用ID")
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		Unauthorized(c, "未登录")
		return
	}

	if err := h.svc.StopApp(context.Background(), uint(appID), userID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, nil)
}

// RestartApp 重启应用
// @Summary 重启应用
// @Description 重启指定的应用
// @Tags 应用
// @Produce json
// @Security Bearer
// @Param id path int true "应用ID"
// @Success 200 {object} Response "重启成功"
// @Failure 401 {object} Response "未授权"
// @Failure 404 {object} Response "应用不存在"
// @Router /apps/{id}/restart [post]
func (h *AppHandler) RestartApp(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		BadRequest(c, "无效的应用ID")
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		Unauthorized(c, "未登录")
		return
	}

	if err := h.svc.RestartApp(context.Background(), uint(appID), userID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, nil)
}

// GetAppLogs 获取应用日志
// @Summary 获取应用日志
// @Description 获取指定应用的容器日志
// @Tags 应用
// @Produce json
// @Security Bearer
// @Param id path int true "应用ID"
// @Param lines query int false "日志行数" default(100)
// @Success 200 {object} Response{data=AppLogsResponse} "成功"
// @Failure 401 {object} Response "未授权"
// @Failure 404 {object} Response "应用不存在"
// @Router /apps/{id}/logs [get]
func (h *AppHandler) GetAppLogs(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		BadRequest(c, "无效的应用ID")
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		Unauthorized(c, "未登录")
		return
	}

	lines := int64(100)
	if linesStr := c.Query("lines"); linesStr != "" {
		if l, err := strconv.ParseInt(linesStr, 10, 64); err == nil && l > 0 {
			lines = l
		}
	}

	logs, err := h.svc.GetAppLogs(context.Background(), uint(appID), userID, lines)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, AppLogsResponse{Logs: logs})
}

// RegisterAppRoutes 注册应用相关路由
func RegisterAppRoutes(r *gin.RouterGroup) {
	h := NewAppHandler()
	apps := r.Group("/apps")
	{
		apps.POST("", h.CreateApp)
		apps.GET("", h.GetApps)
		apps.GET("/:id", h.GetApp)
		apps.DELETE("/:id", h.DeleteApp)
		apps.POST("/:id/start", h.StartApp)
		apps.POST("/:id/stop", h.StopApp)
		apps.POST("/:id/restart", h.RestartApp)
		apps.GET("/:id/logs", h.GetAppLogs)
	}
}
