package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyOAuth2Environment(t *testing.T) {
	t.Setenv("ASTRO_OAUTH2_AUTHENTIK_CLIENT_ID", "astro-test-web")
	t.Setenv("ASTRO_OAUTH2_AUTHENTIK_CLIENT_SECRET", "environment-secret")
	t.Setenv("ASTRO_OAUTH2_AUTHENTIK_REDIRECT_URL", "https://astro-test.bytcloud.org/oauth2/bytcloudauth/callback")

	cfg := &Config{OAuth2: OAuth2Config{Providers: map[string]OAuth2ProviderConfig{
		"authentik": {
			ClientID:     "astro-web",
			ClientSecret: "file-secret",
			RedirectURL:  "http://localhost:5173/oauth2/bytcloudauth/callback",
		},
	}}}
	if err := applyDevelopmentOAuth2Environment(cfg); err != nil {
		t.Fatalf("应用 OAuth2 环境变量覆盖失败: %v", err)
	}

	provider := cfg.OAuth2.Providers["authentik"]
	if provider.ClientID != "astro-test-web" || provider.ClientSecret != "environment-secret" || provider.RedirectURL != "https://astro-test.bytcloud.org/oauth2/bytcloudauth/callback" {
		t.Fatalf("OAuth2 environment override was not applied: %#v", provider)
	}
}

func TestDevelopmentOAuth2EnvironmentRejectsPartialOverride(t *testing.T) {
	t.Setenv("ASTRO_OAUTH2_AUTHENTIK_CLIENT_ID", "astro-test-web")
	t.Setenv("ASTRO_OAUTH2_AUTHENTIK_CLIENT_SECRET", "")
	t.Setenv("ASTRO_OAUTH2_AUTHENTIK_REDIRECT_URL", "")

	cfg := &Config{OAuth2: OAuth2Config{Providers: map[string]OAuth2ProviderConfig{
		authentikProviderKey: {},
	}}}
	if err := applyDevelopmentOAuth2Environment(cfg); err == nil {
		t.Fatal("不完整的开发 OAuth2 覆盖未被拒绝")
	}
}

func TestLoadDevelopmentConfigPreservesDefaults(t *testing.T) {
	t.Setenv("ASTRO_RUNTIME_ENV", "")
	t.Setenv("ASTRO_OAUTH2_AUTHENTIK_CLIENT_ID", "")
	t.Setenv("ASTRO_OAUTH2_AUTHENTIK_CLIENT_SECRET", "")
	t.Setenv("ASTRO_OAUTH2_AUTHENTIK_REDIRECT_URL", "")

	configPath := writeTestConfig(t)
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("加载开发配置失败: %v", err)
	}
	if cfg.Server.Mode != "debug" || cfg.JWT.Secret != defaultJWTSecret || cfg.Database.Password != "" {
		t.Fatalf("开发配置默认值被意外修改: %#v", cfg)
	}
}

func TestLoadRuntimeConfig(t *testing.T) {
	for _, runtimeEnvironment := range []string{runtimeEnvironmentTest, runtimeEnvironmentProduction} {
		t.Run(runtimeEnvironment, func(t *testing.T) {
			kubeconfigPath := writeKubeconfig(t, "apiVersion: v1\nkind: Config\n")
			setValidRuntimeEnvironment(t, runtimeEnvironment, kubeconfigPath)

			cfg, err := Load(writeTestConfig(t))
			if err != nil {
				t.Fatalf("加载 %s 运行时配置失败: %v", runtimeEnvironment, err)
			}
			provider := cfg.OAuth2.Providers[authentikProviderKey]
			if cfg.Server.Port != 8080 || cfg.Server.Mode != "release" || cfg.Database.Host != "mariadb" || cfg.Database.Password != "runtime-database-password" {
				t.Fatalf("运行时基础配置覆盖不完整: %#v", cfg)
			}
			if cfg.Kubernetes.Kubeconfig != kubeconfigPath || provider.ClientSecret != "runtime-oauth-client-secret" {
				t.Fatalf("运行时秘密配置覆盖不完整: %#v", cfg)
			}
		})
	}
}

func TestLoadRuntimeConfigRejectsInvalidValues(t *testing.T) {
	testCases := []struct {
		name        string
		configure   func(t *testing.T)
		errorString string
	}{
		{
			name: "缺少数据库密码",
			configure: func(t *testing.T) {
				t.Setenv("ASTRO_DATABASE_PASSWORD", "")
			},
			errorString: "ASTRO_DATABASE_PASSWORD",
		},
		{
			name: "使用默认 JWT Secret",
			configure: func(t *testing.T) {
				t.Setenv("ASTRO_JWT_SECRET", defaultJWTSecret)
			},
			errorString: "JWT Secret",
		},
		{
			name: "使用开发回调地址",
			configure: func(t *testing.T) {
				t.Setenv("ASTRO_OAUTH2_AUTHENTIK_REDIRECT_URL", "http://localhost:5173/oauth2/bytcloudauth/callback")
			},
			errorString: "回调地址",
		},
		{
			name: "使用错误的 OAuth2 Client ID",
			configure: func(t *testing.T) {
				t.Setenv("ASTRO_OAUTH2_AUTHENTIK_CLIENT_ID", "astro-web")
			},
			errorString: "Client ID",
		},
		{
			name: "kubeconfig 不存在",
			configure: func(t *testing.T) {
				t.Setenv("ASTRO_KUBERNETES_KUBECONFIG", filepath.Join(t.TempDir(), "missing-kubeconfig"))
			},
			errorString: "kubeconfig 不可用",
		},
		{
			name: "kubeconfig 为空",
			configure: func(t *testing.T) {
				t.Setenv("ASTRO_KUBERNETES_KUBECONFIG", writeKubeconfig(t, ""))
			},
			errorString: "非空普通文件",
		},
		{
			name: "服务模式无效",
			configure: func(t *testing.T) {
				t.Setenv("ASTRO_SERVER_MODE", "debug")
			},
			errorString: "release 模式",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			setValidRuntimeEnvironment(t, runtimeEnvironmentTest, writeKubeconfig(t, "apiVersion: v1\nkind: Config\n"))
			testCase.configure(t)

			_, err := Load(writeTestConfig(t))
			if err == nil {
				t.Fatal("无效运行时配置未被拒绝")
			}
			if !strings.Contains(err.Error(), testCase.errorString) {
				t.Fatalf("错误信息 %q 未包含 %q", err.Error(), testCase.errorString)
			}
		})
	}
}

func TestLoadRejectsInvalidRuntimeEnvironment(t *testing.T) {
	t.Setenv("ASTRO_RUNTIME_ENV", "staging")

	_, err := Load(writeTestConfig(t))
	if err == nil || !strings.Contains(err.Error(), "ASTRO_RUNTIME_ENV") {
		t.Fatalf("无效运行环境未被拒绝: %v", err)
	}
}

func setValidRuntimeEnvironment(t *testing.T, runtimeEnvironment, kubeconfigPath string) {
	t.Helper()
	redirectURL := "https://astro-test.bytcloud.org/oauth2/bytcloudauth/callback"
	clientID := "astro-test-web"
	if runtimeEnvironment == runtimeEnvironmentProduction {
		redirectURL = "https://astro.bytcloud.org/oauth2/bytcloudauth/callback"
		clientID = "astro-web"
	}

	environmentValues := map[string]string{
		"ASTRO_RUNTIME_ENV":                    runtimeEnvironment,
		"ASTRO_SERVER_PORT":                    "8080",
		"ASTRO_SERVER_MODE":                    "release",
		"ASTRO_DATABASE_HOST":                  "mariadb",
		"ASTRO_DATABASE_PORT":                  "3306",
		"ASTRO_DATABASE_USER":                  "astro_runtime",
		"ASTRO_DATABASE_PASSWORD":              "runtime-database-password",
		"ASTRO_DATABASE_DBNAME":                "astro_runtime",
		"ASTRO_DATABASE_CHARSET":               "utf8mb4",
		"ASTRO_JWT_SECRET":                     "runtime-jwt-secret-with-at-least-32-bytes",
		"ASTRO_KUBERNETES_KUBECONFIG":          kubeconfigPath,
		"ASTRO_LOG_LEVEL":                      "info",
		"ASTRO_LOG_FILE":                       "/tmp/astro.log",
		"ASTRO_OAUTH2_AUTHENTIK_CLIENT_ID":     clientID,
		"ASTRO_OAUTH2_AUTHENTIK_CLIENT_SECRET": "runtime-oauth-client-secret",
		"ASTRO_OAUTH2_AUTHENTIK_REDIRECT_URL":  redirectURL,
	}
	for name, value := range environmentValues {
		t.Setenv(name, value)
	}
}

func writeKubeconfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("写入测试 kubeconfig 失败: %v", err)
	}
	return path
}

func writeTestConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := `server:
  port: 8080
  mode: debug
database:
  host: localhost
  port: 3306
  user: root
  password: ""
  dbname: astro
  charset: utf8mb4
jwt:
  secret: astro-secret-key
  expire: 24h
log:
  level: debug
  file: logs/astro.log
  max_size: 100
  max_backups: 10
  max_age: 30
  compress: true
kubernetes:
  kubeconfig: ""
oauth2:
  providers:
    authentik:
      enabled: true
      client_id: astro-web
      client_secret: ""
      redirect_url: http://localhost:5173/oauth2/bytcloudauth/callback
      auth_url: https://auth.bytcloud.org/application/o/authorize/
      token_url: https://auth.bytcloud.org/application/o/token/
      userinfo_url: https://auth.bytcloud.org/application/o/userinfo/
      scopes:
        - openid
        - email
        - profile
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	return path
}
