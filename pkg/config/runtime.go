package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	runtimeEnvironmentTest       = "test"
	runtimeEnvironmentProduction = "production"
	authentikProviderKey         = "authentik"
	defaultJWTSecret             = "astro-secret-key"
)

type runtimeEnvironmentValues struct {
	serverPort        int
	serverMode        string
	databaseHost      string
	databasePort      int
	databaseUser      string
	databasePassword  string
	databaseName      string
	databaseCharset   string
	jwtSecret         string
	kubeconfig        string
	logLevel          string
	logFile           string
	oauthClientID     string
	oauthClientSecret string
	oauthRedirectURL  string
}

func applyEnvironment(cfg *Config) (string, error) {
	runtimeEnvironment := strings.TrimSpace(os.Getenv("ASTRO_RUNTIME_ENV"))
	if runtimeEnvironment == "" {
		if err := applyDevelopmentOAuth2Environment(cfg); err != nil {
			return "", err
		}
		return "", nil
	}
	if runtimeEnvironment != runtimeEnvironmentTest && runtimeEnvironment != runtimeEnvironmentProduction {
		return "", fmt.Errorf("ASTRO_RUNTIME_ENV 仅支持 test 或 production")
	}

	values, err := readRuntimeEnvironmentValues()
	if err != nil {
		return "", err
	}
	if err := values.apply(cfg); err != nil {
		return "", err
	}
	return runtimeEnvironment, nil
}

func readRuntimeEnvironmentValues() (runtimeEnvironmentValues, error) {
	serverPort, err := requiredEnvironmentPort("ASTRO_SERVER_PORT")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}
	databasePort, err := requiredEnvironmentPort("ASTRO_DATABASE_PORT")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}

	serverMode, err := requiredEnvironmentValue("ASTRO_SERVER_MODE")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}
	databaseHost, err := requiredEnvironmentValue("ASTRO_DATABASE_HOST")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}
	databaseUser, err := requiredEnvironmentValue("ASTRO_DATABASE_USER")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}
	databasePassword, err := requiredEnvironmentValue("ASTRO_DATABASE_PASSWORD")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}
	databaseName, err := requiredEnvironmentValue("ASTRO_DATABASE_DBNAME")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}
	databaseCharset, err := requiredEnvironmentValue("ASTRO_DATABASE_CHARSET")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}
	jwtSecret, err := requiredEnvironmentValue("ASTRO_JWT_SECRET")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}
	kubeconfig, err := requiredEnvironmentValue("ASTRO_KUBERNETES_KUBECONFIG")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}
	logLevel, err := requiredEnvironmentValue("ASTRO_LOG_LEVEL")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}
	logFile, err := requiredEnvironmentValue("ASTRO_LOG_FILE")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}
	oauthClientID, err := requiredEnvironmentValue("ASTRO_OAUTH2_AUTHENTIK_CLIENT_ID")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}
	oauthClientSecret, err := requiredEnvironmentValue("ASTRO_OAUTH2_AUTHENTIK_CLIENT_SECRET")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}
	oauthRedirectURL, err := requiredEnvironmentValue("ASTRO_OAUTH2_AUTHENTIK_REDIRECT_URL")
	if err != nil {
		return runtimeEnvironmentValues{}, err
	}

	return runtimeEnvironmentValues{
		serverPort:        serverPort,
		serverMode:        strings.TrimSpace(serverMode),
		databaseHost:      strings.TrimSpace(databaseHost),
		databasePort:      databasePort,
		databaseUser:      strings.TrimSpace(databaseUser),
		databasePassword:  databasePassword,
		databaseName:      strings.TrimSpace(databaseName),
		databaseCharset:   strings.TrimSpace(databaseCharset),
		jwtSecret:         jwtSecret,
		kubeconfig:        strings.TrimSpace(kubeconfig),
		logLevel:          strings.TrimSpace(logLevel),
		logFile:           strings.TrimSpace(logFile),
		oauthClientID:     strings.TrimSpace(oauthClientID),
		oauthClientSecret: oauthClientSecret,
		oauthRedirectURL:  strings.TrimSpace(oauthRedirectURL),
	}, nil
}

func (values runtimeEnvironmentValues) apply(cfg *Config) error {
	provider, ok := cfg.OAuth2.Providers[authentikProviderKey]
	if !ok {
		return fmt.Errorf("OAuth2 配置缺少 BytCloud Auth Provider")
	}

	cfg.Server.Port = values.serverPort
	cfg.Server.Mode = values.serverMode
	cfg.Database.Host = values.databaseHost
	cfg.Database.Port = values.databasePort
	cfg.Database.User = values.databaseUser
	cfg.Database.Password = values.databasePassword
	cfg.Database.DBName = values.databaseName
	cfg.Database.Charset = values.databaseCharset
	cfg.JWT.Secret = values.jwtSecret
	cfg.Kubernetes.Kubeconfig = values.kubeconfig
	cfg.Log.Level = values.logLevel
	cfg.Log.File = values.logFile
	provider.ClientID = values.oauthClientID
	provider.ClientSecret = values.oauthClientSecret
	provider.RedirectURL = values.oauthRedirectURL
	cfg.OAuth2.Providers[authentikProviderKey] = provider
	return nil
}

func requiredEnvironmentValue(name string) (string, error) {
	value, exists := os.LookupEnv(name)
	if !exists || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("缺少运行时环境变量 %s", name)
	}
	return value, nil
}

func requiredEnvironmentPort(name string) (int, error) {
	value, err := requiredEnvironmentValue(name)
	if err != nil {
		return 0, err
	}
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("运行时环境变量 %s 必须是有效端口", name)
	}
	return port, nil
}

func applyDevelopmentOAuth2Environment(cfg *Config) error {
	provider, ok := cfg.OAuth2.Providers[authentikProviderKey]
	if !ok {
		return nil
	}

	clientID, clientIDSet := os.LookupEnv("ASTRO_OAUTH2_AUTHENTIK_CLIENT_ID")
	clientSecret, clientSecretSet := os.LookupEnv("ASTRO_OAUTH2_AUTHENTIK_CLIENT_SECRET")
	redirectURL, redirectURLSet := os.LookupEnv("ASTRO_OAUTH2_AUTHENTIK_REDIRECT_URL")
	if !clientIDSet && !clientSecretSet && !redirectURLSet {
		return nil
	}
	if strings.TrimSpace(clientID) == "" && strings.TrimSpace(clientSecret) == "" && strings.TrimSpace(redirectURL) == "" {
		return nil
	}
	if !clientIDSet || strings.TrimSpace(clientID) == "" || !clientSecretSet || strings.TrimSpace(clientSecret) == "" || !redirectURLSet || strings.TrimSpace(redirectURL) == "" {
		return fmt.Errorf("开发环境 OAuth2 覆盖必须同时提供 Client ID、Client Secret 和回调地址")
	}

	if clientID != "" {
		provider.ClientID = clientID
	}
	if clientSecret != "" {
		provider.ClientSecret = clientSecret
	}
	if redirectURL != "" {
		provider.RedirectURL = redirectURL
	}
	cfg.OAuth2.Providers[authentikProviderKey] = provider
	return nil
}

// ValidateRuntime 校验测试和生产环境的关键运行时配置。
func ValidateRuntime(cfg *Config, runtimeEnvironment string) error {
	if runtimeEnvironment == "" {
		return nil
	}
	if cfg == nil {
		return fmt.Errorf("运行时配置不能为空")
	}
	if runtimeEnvironment != runtimeEnvironmentTest && runtimeEnvironment != runtimeEnvironmentProduction {
		return fmt.Errorf("运行环境无效")
	}
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("服务端口无效")
	}
	if cfg.Server.Mode != "release" {
		return fmt.Errorf("测试和生产环境必须使用 release 模式")
	}
	if strings.TrimSpace(cfg.Database.Host) == "" || strings.TrimSpace(cfg.Database.User) == "" || strings.TrimSpace(cfg.Database.DBName) == "" || strings.TrimSpace(cfg.Database.Charset) == "" {
		return fmt.Errorf("数据库连接配置不完整")
	}
	if cfg.Database.Port < 1 || cfg.Database.Port > 65535 {
		return fmt.Errorf("数据库端口无效")
	}
	if strings.TrimSpace(cfg.Database.Password) == "" {
		return fmt.Errorf("数据库密码不能为空")
	}
	if err := validateJWTSecret(cfg.JWT.Secret); err != nil {
		return err
	}
	if err := validateKubeconfig(cfg.Kubernetes.Kubeconfig); err != nil {
		return err
	}
	if err := validateLogConfig(cfg.Log); err != nil {
		return err
	}
	return validateOAuth2Runtime(cfg.OAuth2, runtimeEnvironment)
}

func validateJWTSecret(secret string) error {
	trimmedSecret := strings.TrimSpace(secret)
	if trimmedSecret == "" || trimmedSecret == defaultJWTSecret || len(trimmedSecret) < 32 || isPlaceholderSecret(trimmedSecret) {
		return fmt.Errorf("JWT Secret 必须使用至少 32 字节的非默认值")
	}
	return nil
}

func validateKubeconfig(path string) error {
	if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) {
		return fmt.Errorf("kubernetes kubeconfig 必须是非空绝对路径")
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("kubernetes kubeconfig 不可用: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return fmt.Errorf("kubernetes kubeconfig 必须是非空普通文件")
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("kubernetes kubeconfig 不可读: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("关闭 kubernetes kubeconfig 失败: %w", err)
	}
	return nil
}

func validateLogConfig(logConfig LogConfig) error {
	switch logConfig.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("日志级别无效")
	}
	if strings.TrimSpace(logConfig.File) == "" || !filepath.IsAbs(logConfig.File) {
		return fmt.Errorf("日志文件必须是非空绝对路径")
	}
	return nil
}

func validateOAuth2Runtime(oauthConfig OAuth2Config, runtimeEnvironment string) error {
	provider, ok := oauthConfig.Providers[authentikProviderKey]
	if !ok || !provider.Enabled {
		return fmt.Errorf("BytCloud Auth Provider 必须启用")
	}
	if strings.TrimSpace(provider.ClientID) == "" {
		return fmt.Errorf("BytCloud Auth Client ID 不能为空")
	}
	expectedClientID := "astro-test-web"
	if runtimeEnvironment == runtimeEnvironmentProduction {
		expectedClientID = "astro-web"
	}
	if provider.ClientID != expectedClientID {
		return fmt.Errorf("BytCloud Auth Client ID 与运行环境不匹配")
	}
	if strings.TrimSpace(provider.ClientSecret) == "" || isPlaceholderSecret(provider.ClientSecret) {
		return fmt.Errorf("BytCloud Auth Client Secret 不能为空或示例值")
	}

	expectedRedirectURL := "https://astro-test.bytcloud.org/oauth2/bytcloudauth/callback"
	if runtimeEnvironment == runtimeEnvironmentProduction {
		expectedRedirectURL = "https://astro.bytcloud.org/oauth2/bytcloudauth/callback"
	}
	if provider.RedirectURL != expectedRedirectURL {
		return fmt.Errorf("BytCloud Auth 回调地址与运行环境不匹配")
	}
	if err := validateHTTPSURL(provider.AuthURL); err != nil {
		return fmt.Errorf("BytCloud Auth 授权地址无效: %w", err)
	}
	if err := validateHTTPSURL(provider.TokenURL); err != nil {
		return fmt.Errorf("BytCloud Auth Token 地址无效: %w", err)
	}
	if err := validateHTTPSURL(provider.UserInfoURL); err != nil {
		return fmt.Errorf("BytCloud Auth UserInfo 地址无效: %w", err)
	}
	return nil
}

func validateHTTPSURL(value string) error {
	parsedURL, err := url.ParseRequestURI(value)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return fmt.Errorf("必须使用完整 HTTPS URL")
	}
	return nil
}

func isPlaceholderSecret(value string) bool {
	normalizedValue := strings.ToLower(strings.TrimSpace(value))
	for _, placeholder := range []string{"change-me", "changeme", "example-secret", "your-secret", "client-secret"} {
		if normalizedValue == placeholder {
			return true
		}
	}
	return false
}
