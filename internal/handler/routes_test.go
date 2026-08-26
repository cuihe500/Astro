package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cuihe500/astro/pkg/errcode"
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

func TestProjectAndAppRoutesRejectInvalidInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	RegisterProjectRoutes(api)
	RegisterAppRoutes(api)

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/api/v1/projects/not-a-number"},
		{method: http.MethodPost, path: "/api/v1/projects", body: `{"name":"   "}`},
		{method: http.MethodGet, path: "/api/v1/projects/1/apps/0"},
		{method: http.MethodPost, path: "/api/v1/projects/1/apps", body: `{"name":"INVALID","image":"nginx:latest","replicas":1}`},
		{method: http.MethodPost, path: "/api/v1/projects/1/apps", body: `{"name":"demo","image":"nginx:latest","replicas":1,"hostNetwork":true}`},
		{method: http.MethodPost, path: "/api/v1/projects/1/apps", body: `{"name":"demo","image":"nginx:latest","replicas":1}{}`},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, bytes.NewBufferString(test.body))
			request.Header.Set("Content-Type", "application/json")
			responseRecorder := httptest.NewRecorder()
			router.ServeHTTP(responseRecorder, request)
			var response Response
			if err := json.Unmarshal(responseRecorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("解析错误响应失败: %v", err)
			}
			if response.Code != errcode.ErrBadRequest.Int() {
				t.Fatalf("错误码 = %d, want %d", response.Code, errcode.ErrBadRequest.Int())
			}
		})
	}
}
