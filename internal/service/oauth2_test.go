package service

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/cuihe500/astro/pkg/config"
	"github.com/cuihe500/astro/pkg/errcode"
)

func TestResolveOAuth2Provider(t *testing.T) {
	providerKey, err := resolveOAuth2Provider(bytCloudAuthAlias)
	if err != nil {
		t.Fatalf("解析公开别名失败: %v", err)
	}
	if providerKey != bytCloudAuthProviderKey {
		t.Fatalf("内部 Provider 键错误: got %q, want %q", providerKey, bytCloudAuthProviderKey)
	}

	_, err = resolveOAuth2Provider(bytCloudAuthProviderKey)
	if err == nil {
		t.Fatal("内部 Provider 键不能作为公开别名")
	}
	coded, ok := err.(*errcode.Error)
	if !ok || coded.Code != errcode.ErrBadRequest {
		t.Fatalf("内部 Provider 键错误码错误: %#v", err)
	}
	if strings.Contains(strings.ToLower(err.Error()), bytCloudAuthProviderKey) {
		t.Fatalf("错误消息泄露内部 Provider 键: %q", err.Error())
	}
}

func TestOAuth2DatabaseErrorDoesNotLeakInternalProviderKey(t *testing.T) {
	err := newOAuth2DatabaseError()
	if err.Code != errcode.ErrDatabase {
		t.Fatalf("数据库错误码错误: got %d, want %d", err.Code, errcode.ErrDatabase)
	}
	if strings.Contains(strings.ToLower(err.Error()), bytCloudAuthProviderKey) {
		t.Fatalf("数据库错误泄露内部 Provider 键: %q", err.Error())
	}
}

func TestBuildAuthURLUsesPublicAliasAndInternalConfig(t *testing.T) {
	previousConfig := config.GlobalConfig
	t.Cleanup(func() {
		config.GlobalConfig = previousConfig
	})

	config.GlobalConfig = &config.Config{
		JWT: config.JWTConfig{Secret: "secret"},
		OAuth2: config.OAuth2Config{Providers: map[string]config.OAuth2ProviderConfig{
			bytCloudAuthProviderKey: {
				Enabled:      true,
				ClientID:     "client",
				ClientSecret: "secret",
				RedirectURL:  "https://astro.bytcloud.org/oauth2/bytcloudauth/callback",
				AuthURL:      "https://identity.example/authorize",
				TokenURL:     "https://identity.example/token",
				UserInfoURL:  "https://identity.example/userinfo",
				Scopes:       []string{"openid"},
			},
		}},
	}

	authURL, err := NewOAuth2Service().BuildAuthURL(bytCloudAuthAlias)
	if err != nil {
		t.Fatalf("生成授权 URL 失败: %v", err)
	}
	parsedURL, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("解析授权 URL 失败: %v", err)
	}
	if got := parsedURL.Query().Get("redirect_uri"); got != "https://astro.bytcloud.org/oauth2/bytcloudauth/callback" {
		t.Fatalf("redirect_uri 错误: got %q", got)
	}
	if err := validateOAuth2State(parsedURL.Query().Get("state"), bytCloudAuthAlias, "secret", time.Now()); err != nil {
		t.Fatalf("授权 URL 的 state 未绑定公开别名: %v", err)
	}
}

func TestOAuth2State(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	state, err := generateOAuth2State(bytCloudAuthAlias, "secret", now)
	if err != nil {
		t.Fatalf("生成 state 失败: %v", err)
	}

	if err := validateOAuth2State(state, bytCloudAuthAlias, "secret", now.Add(time.Minute)); err != nil {
		t.Fatalf("校验 state 失败: %v", err)
	}

	if err := validateOAuth2State(state+"x", bytCloudAuthAlias, "secret", now); err == nil {
		t.Fatal("篡改 state 应该失败")
	}

	if err := validateOAuth2State(state, "other", "secret", now); err == nil {
		t.Fatal("provider 不匹配应该失败")
	}

	if err := validateOAuth2State(state, bytCloudAuthAlias, "secret", now.Add(oauth2StateTTL+time.Second)); err == nil {
		t.Fatal("过期 state 应该失败")
	}
}

func TestOAuth2StateUsesPublicAlias(t *testing.T) {
	state, err := generateOAuth2State(bytCloudAuthAlias, "secret", time.Now())
	if err != nil {
		t.Fatalf("生成 state 失败: %v", err)
	}

	parts := strings.Split(state, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		t.Fatalf("state 格式错误: %q", state)
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("解码 state 失败: %v", err)
	}
	var payload oauth2StatePayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		t.Fatalf("解析 state 失败: %v", err)
	}
	if payload.Provider != bytCloudAuthAlias {
		t.Fatalf("state Provider 错误: got %q, want %q", payload.Provider, bytCloudAuthAlias)
	}
	if strings.Contains(strings.ToLower(string(payloadJSON)), bytCloudAuthProviderKey) {
		t.Fatalf("state 泄露内部 Provider 键: %q", payloadJSON)
	}
}
