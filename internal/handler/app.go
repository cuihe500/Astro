package handler

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/cuihe500/astro/internal/service"
	"github.com/gin-gonic/gin"
)

var appNamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)

// AppHandler 应用处理器
type AppHandler struct {
	svc *service.AppService
}

// NewAppHandler 创建应用处理器
func NewAppHandler() *AppHandler {
	return &AppHandler{svc: service.NewAppService()}
}

// CreateAppRequest 创建应用请求
type CreateAppRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=63" example:"my-nginx"`
	Image    string `json:"image" binding:"required,max=256" example:"nginx:latest"`
	Replicas int    `json:"replicas" binding:"required,min=1,max=10" example:"2"`
	Port     int    `json:"port" binding:"omitempty,min=1,max=65535" example:"80"`
}

// AppLogsResponse 日志响应
type AppLogsResponse struct {
	Logs string `json:"logs"`
}

// CreateApp 创建应用
// @Summary 创建应用
// @Description 在指定项目中创建一个新的容器应用
// @Tags 应用
// @Accept json
// @Produce json
// @Security Bearer
// @Param project_id path int true "项目ID"
// @Param request body CreateAppRequest true "应用信息"
// @Success 200 {object} Response "创建成功"
// @Failure 400 {object} Response "参数错误"
// @Failure 401 {object} Response "未授权"
// @Router /projects/{project_id}/apps [post]
func (h *AppHandler) CreateApp(c *gin.Context) {
	projectID, ok := parsePositiveID(c, "project_id", "项目ID")
	if !ok {
		return
	}
	var req CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Image = strings.TrimSpace(req.Image)
	if !appNamePattern.MatchString(req.Name) || req.Image == "" {
		BadRequest(c, "应用名称或容器镜像无效")
		return
	}
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}

	app, err := h.svc.CreateApp(c.Request.Context(), service.CreateAppRequest{
		Name: req.Name, Image: req.Image, Replicas: req.Replicas, Port: req.Port,
		ProjectID: projectID, UserID: userID,
	})
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, app)
}

// GetApps 获取应用列表
// @Summary 获取应用列表
// @Description 获取指定项目的所有应用
// @Tags 应用
// @Produce json
// @Security Bearer
// @Param project_id path int true "项目ID"
// @Success 200 {object} Response "成功"
// @Failure 401 {object} Response "未授权"
// @Router /projects/{project_id}/apps [get]
func (h *AppHandler) GetApps(c *gin.Context) {
	projectID, ok := parsePositiveID(c, "project_id", "项目ID")
	if !ok {
		return
	}
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}
	apps, err := h.svc.GetApps(c.Request.Context(), projectID, userID)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, apps)
}

// GetApp 获取应用详情
// @Summary 获取应用详情
// @Description 获取指定项目内应用的详细信息
// @Tags 应用
// @Produce json
// @Security Bearer
// @Param project_id path int true "项目ID"
// @Param id path int true "应用ID"
// @Success 200 {object} Response "成功"
// @Failure 401 {object} Response "未授权"
// @Failure 404 {object} Response "应用不存在"
// @Router /projects/{project_id}/apps/{id} [get]
func (h *AppHandler) GetApp(c *gin.Context) {
	projectID, appID, userID, ok := appRequestIDs(c)
	if !ok {
		return
	}
	app, err := h.svc.GetApp(c.Request.Context(), projectID, appID, userID)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, app)
}

// DeleteApp 删除应用
// @Summary 删除应用
// @Description 删除指定项目内的应用
// @Tags 应用
// @Produce json
// @Security Bearer
// @Param project_id path int true "项目ID"
// @Param id path int true "应用ID"
// @Success 200 {object} Response "删除成功"
// @Failure 401 {object} Response "未授权"
// @Failure 404 {object} Response "应用不存在"
// @Router /projects/{project_id}/apps/{id} [delete]
func (h *AppHandler) DeleteApp(c *gin.Context) {
	projectID, appID, userID, ok := appRequestIDs(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteApp(c.Request.Context(), projectID, appID, userID); err != nil {
		HandleError(c, err)
		return
	}
	Success(c, nil)
}

// StartApp 启动应用
// @Summary 启动应用
// @Description 启动指定项目内的应用
// @Tags 应用
// @Produce json
// @Security Bearer
// @Param project_id path int true "项目ID"
// @Param id path int true "应用ID"
// @Success 200 {object} Response "启动成功"
// @Router /projects/{project_id}/apps/{id}/start [post]
func (h *AppHandler) StartApp(c *gin.Context) {
	projectID, appID, userID, ok := appRequestIDs(c)
	if !ok {
		return
	}
	if err := h.svc.StartApp(c.Request.Context(), projectID, appID, userID); err != nil {
		HandleError(c, err)
		return
	}
	Success(c, nil)
}

// StopApp 停止应用
// @Summary 停止应用
// @Description 停止指定项目内的应用
// @Tags 应用
// @Produce json
// @Security Bearer
// @Param project_id path int true "项目ID"
// @Param id path int true "应用ID"
// @Success 200 {object} Response "停止成功"
// @Router /projects/{project_id}/apps/{id}/stop [post]
func (h *AppHandler) StopApp(c *gin.Context) {
	projectID, appID, userID, ok := appRequestIDs(c)
	if !ok {
		return
	}
	if err := h.svc.StopApp(c.Request.Context(), projectID, appID, userID); err != nil {
		HandleError(c, err)
		return
	}
	Success(c, nil)
}

// RestartApp 重启应用
// @Summary 重启应用
// @Description 重启指定项目内的应用
// @Tags 应用
// @Produce json
// @Security Bearer
// @Param project_id path int true "项目ID"
// @Param id path int true "应用ID"
// @Success 200 {object} Response "重启成功"
// @Router /projects/{project_id}/apps/{id}/restart [post]
func (h *AppHandler) RestartApp(c *gin.Context) {
	projectID, appID, userID, ok := appRequestIDs(c)
	if !ok {
		return
	}
	if err := h.svc.RestartApp(c.Request.Context(), projectID, appID, userID); err != nil {
		HandleError(c, err)
		return
	}
	Success(c, nil)
}

// GetAppLogs 获取应用日志
// @Summary 获取应用日志
// @Description 获取指定项目内应用的容器日志
// @Tags 应用
// @Produce json
// @Security Bearer
// @Param project_id path int true "项目ID"
// @Param id path int true "应用ID"
// @Param lines query int false "日志行数" default(100)
// @Success 200 {object} Response{data=AppLogsResponse} "成功"
// @Router /projects/{project_id}/apps/{id}/logs [get]
func (h *AppHandler) GetAppLogs(c *gin.Context) {
	projectID, appID, userID, ok := appRequestIDs(c)
	if !ok {
		return
	}
	lines := int64(100)
	if value := c.Query("lines"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed < 1 || parsed > 1000 {
			BadRequest(c, "日志行数必须是 1-1000 的整数")
			return
		}
		lines = parsed
	}
	logs, err := h.svc.GetAppLogs(c.Request.Context(), projectID, appID, userID, lines)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, AppLogsResponse{Logs: logs})
}

func appRequestIDs(c *gin.Context) (uint, uint, uint, bool) {
	projectID, ok := parsePositiveID(c, "project_id", "项目ID")
	if !ok {
		return 0, 0, 0, false
	}
	appID, ok := parsePositiveID(c, "id", "应用ID")
	if !ok {
		return 0, 0, 0, false
	}
	userID, ok := authenticatedUserID(c)
	return projectID, appID, userID, ok
}

// RegisterAppRoutes 注册应用相关路由
func RegisterAppRoutes(r *gin.RouterGroup) {
	handler := NewAppHandler()
	apps := r.Group("/projects/:project_id/apps")
	apps.POST("", handler.CreateApp)
	apps.GET("", handler.GetApps)
	apps.GET("/:id", handler.GetApp)
	apps.DELETE("/:id", handler.DeleteApp)
	apps.POST("/:id/start", handler.StartApp)
	apps.POST("/:id/stop", handler.StopApp)
	apps.POST("/:id/restart", handler.RestartApp)
	apps.GET("/:id/logs", handler.GetAppLogs)
}
