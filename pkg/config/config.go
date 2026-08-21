package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Log        LogConfig        `mapstructure:"log"`
	Kubernetes KubernetesConfig `mapstructure:"kubernetes"`
	OAuth2     OAuth2Config     `mapstructure:"oauth2"`
}

// OAuth2Config OAuth2/OIDC 配置
type OAuth2Config struct {
	Providers map[string]OAuth2ProviderConfig `mapstructure:"providers"`
}

// OAuth2ProviderConfig OAuth2/OIDC Provider 配置
type OAuth2ProviderConfig struct {
	Enabled      bool     `mapstructure:"enabled"`
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	RedirectURL  string   `mapstructure:"redirect_url"`
	AuthURL      string   `mapstructure:"auth_url"`
	TokenURL     string   `mapstructure:"token_url"`
	UserInfoURL  string   `mapstructure:"userinfo_url"`
	Scopes       []string `mapstructure:"scopes"`
}

// KubernetesConfig K8s 客户端配置
type KubernetesConfig struct {
	// Kubeconfig 文件路径，留空则使用集群内配置 (InClusterConfig)
	Kubeconfig string `mapstructure:"kubeconfig"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	Charset  string `mapstructure:"charset"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire string `mapstructure:"expire"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`    // 单个日志文件最大大小(MB)
	MaxBackups int    `mapstructure:"max_backups"` // 保留旧日志文件数量
	MaxAge     int    `mapstructure:"max_age"`     // 日志文件保留天数
	Compress   bool   `mapstructure:"compress"`    // 是否压缩归档日志
}

var GlobalConfig *Config

// Load 加载配置文件
func Load(path string) (*Config, error) {
	configLoader := viper.New()
	configLoader.SetConfigFile(path)

	if err := configLoader.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := configLoader.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	runtimeEnvironment, err := applyEnvironment(&cfg)
	if err != nil {
		return nil, fmt.Errorf("加载运行时环境变量失败: %w", err)
	}
	if err := ValidateRuntime(&cfg, runtimeEnvironment); err != nil {
		return nil, fmt.Errorf("运行时配置校验失败: %w", err)
	}

	GlobalConfig = &cfg
	return &cfg, nil
}
