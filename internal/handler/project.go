package handler

import (
	"strings"

	"github.com/cuihe500/astro/internal/service"
	"github.com/gin-gonic/gin"
)

// ProjectHandler 项目处理器
type ProjectHandler struct {
	svc *service.ProjectService
}

// NewProjectHandler 创建项目处理器
func NewProjectHandler() *ProjectHandler {
	return &ProjectHandler{svc: service.NewProjectService()}
}

// CreateProjectRequest 创建项目请求
type CreateProjectRequest struct {
	Name string `json:"name" binding:"required,max=64" example:"个人网站"`
}

// CreateProject 创建项目
// @Summary 创建项目
// @Description 创建项目并建立独立的 Kubernetes Namespace
// @Tags 项目
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body CreateProjectRequest true "项目信息"
// @Success 200 {object} Response "创建成功"
// @Failure 400 {object} Response "参数错误"
// @Failure 401 {object} Response "未授权"
// @Router /projects [post]
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	var req CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "参数错误: "+err.Error())
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		BadRequest(c, "项目名称不能为空")
		return
	}
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}
	project, err := h.svc.CreateProject(c.Request.Context(), userID, req.Name)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, project)
}

// GetProjects 获取项目列表
// @Summary 获取项目列表
// @Description 获取当前用户的所有项目
// @Tags 项目
// @Produce json
// @Security Bearer
// @Success 200 {object} Response "成功"
// @Failure 401 {object} Response "未授权"
// @Router /projects [get]
func (h *ProjectHandler) GetProjects(c *gin.Context) {
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}
	projects, err := h.svc.GetProjects(userID)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, projects)
}

// GetProject 获取项目详情
// @Summary 获取项目详情
// @Description 获取当前用户的指定项目
// @Tags 项目
// @Produce json
// @Security Bearer
// @Param project_id path int true "项目ID"
// @Success 200 {object} Response "成功"
// @Failure 401 {object} Response "未授权"
// @Failure 404 {object} Response "项目不存在"
// @Router /projects/{project_id} [get]
func (h *ProjectHandler) GetProject(c *gin.Context) {
	projectID, ok := parsePositiveID(c, "project_id", "项目ID")
	if !ok {
		return
	}
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}
	project, err := h.svc.GetProject(projectID, userID)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, project)
}

// DeleteProject 删除空项目
// @Summary 删除项目
// @Description 删除空项目及其 Kubernetes Namespace
// @Tags 项目
// @Produce json
// @Security Bearer
// @Param project_id path int true "项目ID"
// @Success 200 {object} Response "删除成功"
// @Failure 401 {object} Response "未授权"
// @Failure 404 {object} Response "项目不存在"
// @Router /projects/{project_id} [delete]
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	projectID, ok := parsePositiveID(c, "project_id", "项目ID")
	if !ok {
		return
	}
	userID, ok := authenticatedUserID(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteProject(c.Request.Context(), projectID, userID); err != nil {
		HandleError(c, err)
		return
	}
	Success(c, nil)
}

// RegisterProjectRoutes 注册项目相关路由
func RegisterProjectRoutes(r *gin.RouterGroup) {
	handler := NewProjectHandler()
	projects := r.Group("/projects")
	projects.POST("", handler.CreateProject)
	projects.GET("", handler.GetProjects)
	projects.GET("/:project_id", handler.GetProject)
	projects.DELETE("/:project_id", handler.DeleteProject)
}
