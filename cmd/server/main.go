package main

import (
	"fmt"
	"os"
	"time"

	"github.com/cuihe500/astro/internal/handler"
	"github.com/cuihe500/astro/internal/k8s"
	"github.com/cuihe500/astro/internal/middleware"
	"github.com/cuihe500/astro/internal/repository"
	"github.com/cuihe500/astro/pkg/config"
	"github.com/cuihe500/astro/pkg/logger"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "github.com/cuihe500/astro/docs"
)

// @title Astro API
// @version 1.0
// @description Astro 容器即服务平台 API 文档

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description 请输入 Bearer {token}

func main() {
	// 加载配置
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化日志
	if err := logger.Init(&cfg.Log); err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "刷新日志失败: %v\n", err)
		}
	}()

	logger.Info("Astro 服务启动中...")

	// 初始化数据库
	if err := repository.Init(&cfg.Database); err != nil {
		logger.Fatal("初始化数据库失败", zap.Error(err))
	}

	// 初始化 K8s 客户端
	if err := k8s.Init(cfg.Kubernetes.Kubeconfig); err != nil {
		logger.Fatal("初始化 K8s 客户端失败", zap.Error(err))
	}
	logger.Info("K8s 客户端初始化成功")

	// 设置运行模式
	gin.SetMode(cfg.Server.Mode)

	// 创建 Gin 引擎。访问日志只记录 Path，避免 OAuth2 code/state 等查询参数进入日志。
	r := gin.New()
	r.Use(gin.LoggerWithFormatter(formatRequestLog), gin.Recovery())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API 路由
	api := r.Group("/api/v1")

	// 公开路由（无需认证）
	handler.RegisterUserRoutes(api)

	// 需要认证的路由
	authApi := api.Group("")
	authApi.Use(middleware.Auth())
	{
		// 项目与应用管理路由
		handler.RegisterProjectRoutes(authApi)
		handler.RegisterAppRoutes(authApi)
	}

	// 启动服务
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	logger.Info("服务启动", zap.String("addr", addr))
	if err := r.Run(addr); err != nil {
		logger.Fatal("启动服务失败", zap.Error(err))
	}
}

func formatRequestLog(params gin.LogFormatterParams) string {
	return fmt.Sprintf("[GIN] %s | %3d | %13v | %15s | %-7s %s\n",
		params.TimeStamp.Format(time.DateTime),
		params.StatusCode,
		params.Latency,
		params.ClientIP,
		params.Method,
		params.Request.URL.Path,
	)
}
