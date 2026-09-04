package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/LittleCurry/go_first_ai/internal/agent"
	"github.com/LittleCurry/go_first_ai/internal/config"
	"github.com/LittleCurry/go_first_ai/internal/models"

	"github.com/gin-gonic/gin"
)

// ChatHandler 聊天处理器
type ChatHandler struct {
	Agent *agent.Agent
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(cfg *config.Config) *ChatHandler {
	return &ChatHandler{
		Agent: agent.NewAgent(cfg),
	}
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Message string `json:"message"`
}

// StreamChat SSE流式聊天（支持POST）
func (h *ChatHandler) StreamChat(c *gin.Context) {
	sessionID := c.Query("session_id")
	//userID := c.Query("user_id")

	// 从POST body获取消息
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	message := req.Message
	if message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}

	if sessionID == "" {
		sessionID = "session_" + time.Now().Format("20060102150405")
	}

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 获取flusher
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// 创建channel
	chunkChan := make(chan string, 100)

	// 异步处理
	go func() {
		defer close(chunkChan)

		history := []models.Message{}

		resp, err := h.Agent.ProcessStream(c.Request.Context(), message, history, chunkChan)
		if err != nil {
			errorData, _ := json.Marshal(map[string]interface{}{
				"type":  "error",
				"error": err.Error(),
			})
			c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", string(errorData))))
			flusher.Flush()
			return
		}

		doneData, _ := json.Marshal(map[string]interface{}{
			"type":    "done",
			"sources": resp.Sources,
		})
		c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", string(doneData))))
		flusher.Flush()
	}()

	// 主循环
	for chunk := range chunkChan {
		chunkData, _ := json.Marshal(map[string]interface{}{
			"type":    "chunk",
			"content": chunk,
		})
		c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", string(chunkData))))
		flusher.Flush()
	}
}
