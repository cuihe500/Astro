package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestLoggerDoesNotWriteQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	router := gin.New()
	router.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Output:    &output,
		Formatter: formatRequestLog,
	}))
	router.GET("/oauth2/bytcloudauth/callback", func(context *gin.Context) {
		context.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/oauth2/bytcloudauth/callback?code=sensitive-code&state=sensitive-state", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	logOutput := output.String()
	if strings.Contains(logOutput, "sensitive-code") || strings.Contains(logOutput, "sensitive-state") || strings.Contains(logOutput, "?") {
		t.Fatalf("访问日志包含敏感查询参数: %q", logOutput)
	}
}
