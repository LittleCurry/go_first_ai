package main

import (
	"log"

	"github.com/LittleCurry/go_first_ai/internal/config"
	"github.com/LittleCurry/go_first_ai/internal/handler"
	"github.com/LittleCurry/go_first_ai/internal/middleware"
	"github.com/LittleCurry/go_first_ai/pkg/logger"
	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	if err := config.LoadConfig(); err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 初始化日志
	logger.Init(config.AppConfig.Server.Debug)

	// 设置Gin模式
	if !config.AppConfig.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建路由
	r := gin.Default()

	// 注册中间件
	r.Use(middleware.CORS())
	r.Use(middleware.Logger())
	r.Use(middleware.RateLimit())

	// 注册路由
	r.GET("/health", handler.HealthCheck)
	r.POST("/api/chat", handler.Chat)
	r.POST("/api/chat/stream", handler.StreamChat)

	// 启动服务
	addr := ":" + config.AppConfig.Server.Port
	logger.Info("Server starting on " + addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
