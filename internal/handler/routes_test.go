package handler

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestProjectRoutesReplaceFlatAppRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	RegisterProjectRoutes(api)
	RegisterAppRoutes(api)

	routes := make(map[string]bool)
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = true
	}
	for _, route := range []string{
		"GET /api/v1/projects",
		"POST /api/v1/projects",
		"GET /api/v1/projects/:project_id",
		"DELETE /api/v1/projects/:project_id",
		"GET /api/v1/projects/:project_id/apps",
		"POST /api/v1/projects/:project_id/apps",
		"GET /api/v1/projects/:project_id/apps/:id",
		"DELETE /api/v1/projects/:project_id/apps/:id",
		"POST /api/v1/projects/:project_id/apps/:id/start",
		"POST /api/v1/projects/:project_id/apps/:id/stop",
		"POST /api/v1/projects/:project_id/apps/:id/restart",
		"GET /api/v1/projects/:project_id/apps/:id/logs",
	} {
		if !routes[route] {
			t.Errorf("缺少路由 %s", route)
		}
	}
	for route := range routes {
		if strings.Contains(route, " /api/v1/apps") {
			t.Fatalf("旧的扁平应用路由仍然存在: %s", route)
		}
	}
}
