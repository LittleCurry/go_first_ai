package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/LittleCurry/go_first_ai/internal/config"
	"github.com/LittleCurry/go_first_ai/internal/handler"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := config.LoadConfig(); err != nil {
		log.Fatal("Failed to load config:", err)
	}

	if !config.AppConfig.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// 静态文件
	r.Static("/static", "./web/static")
	r.GET("/", func(c *gin.Context) {
		c.File("./web/static/index.html")
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	chatHandler := handler.NewChatHandler(config.AppConfig)

	// SSE流式聊天（改为POST）
	r.POST("/api/chat/stream", chatHandler.StreamChat)

	addr := ":" + config.AppConfig.Server.Port
	log.Printf("🚀 Server starting on http://localhost%s", addr)
	log.Printf("📱 Open http://localhost%s/ in your browser", addr)

	go func() {
		if err := r.Run(addr); err != nil {
			log.Fatal("Failed to start server:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")
}
