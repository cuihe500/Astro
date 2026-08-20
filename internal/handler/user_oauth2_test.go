package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cuihe500/astro/pkg/errcode"
	"github.com/gin-gonic/gin"
)

func TestOAuth2InternalProviderAliasIsRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	RegisterUserRoutes(api)

	for _, target := range []string{
		"/api/v1/oauth2/authentik/login",
		"/api/v1/oauth2/authentik/callback?code=code&state=state",
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, target, nil)
		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("%s HTTP 状态错误: got %d, want %d", target, recorder.Code, http.StatusOK)
		}

		var response Response
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("%s 解析响应失败: %v", target, err)
		}
		if response.Code != errcode.ErrBadRequest.Int() {
			t.Fatalf("%s 错误码错误: got %d, want %d", target, response.Code, errcode.ErrBadRequest.Int())
		}
		if strings.Contains(strings.ToLower(response.Message), "authentik") {
			t.Fatalf("%s 错误消息泄露内部 Provider 键: %q", target, response.Message)
		}
	}
}
