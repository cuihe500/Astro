package main

import (
	"fmt"
	"log"

	"github.com/cuihe500/astro/internal/handler"
	"github.com/cuihe500/astro/internal/middleware"
	"github.com/cuihe500/astro/internal/repository"
	"github.com/cuihe500/astro/pkg/config"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

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
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库
	if err := repository.Init(&cfg.Database); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 设置运行模式
	gin.SetMode(cfg.Server.Mode)

	// 创建 Gin 引擎
	r := gin.Default()

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
		// 后续在此处注册需要认证的路由
		// handler.RegisterAppRoutes(authApi)
	}

	// 启动服务
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("服务启动在 %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
}
