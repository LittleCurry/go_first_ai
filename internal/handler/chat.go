package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/LittleCurry/go_first_ai/internal/agent"
	"github.com/LittleCurry/go_first_ai/internal/config"
	"github.com/LittleCurry/go_first_ai/internal/models"
	"github.com/LittleCurry/go_first_ai/internal/session"

	"github.com/gin-gonic/gin"
)

// ChatHandler 聊天处理器
type ChatHandler struct {
	Agent   *agent.Agent
	Session *session.RedisSessionManager
}

// NewChatHandler 创建聊天处理器
func NewChatHandler(cfg *config.Config) *ChatHandler {
	return &ChatHandler{
		Agent:   agent.NewAgent(cfg),
		Session: session.NewRedisSessionManager(cfg),
	}
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Message string `json:"message"`
}

// StreamChat SSE流式聊天
func (h *ChatHandler) StreamChat(c *gin.Context) {
	ctx := context.Background()

	sessionID := c.Query("session_id")
	userID := c.Query("user_id")
	if userID == "" {
		userID = "anonymous"
	}

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
		sessionID = "session_" + time.Now().Format("20060102150405.000")
	}

	// 获取或创建会话
	_, err := h.Session.GetOrCreate(ctx, sessionID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session error"})
		return
	}

	// 获取历史消息（最近10条）
	history, err := h.Session.GetHistory(ctx, sessionID, 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "get history error"})
		return
	}

	// 保存用户消息
	userMsg := models.Message{
		ID:        "msg_" + time.Now().Format("20060102150405.000"),
		SessionID: sessionID,
		Role:      "user",
		Content:   message,
		CreatedAt: time.Now(),
	}
	h.Session.AddMessage(ctx, sessionID, userMsg)

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	chunkChan := make(chan string, 100)

	// 异步处理
	// 在异步处理的 goroutine 中，完成时发送 actions
	go func() {
		defer close(chunkChan)

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

		// 保存AI回复
		aiMsg := models.Message{
			ID:        "msg_" + time.Now().Format("20060102150405.001"),
			SessionID: sessionID,
			Role:      "assistant",
			Content:   resp.Reply,
			Intent:    resp.Intent,
			Sources:   resp.Sources,
			CreatedAt: time.Now(),
		}
		h.Session.AddMessage(ctx, sessionID, aiMsg)

		// 发送完成消息（包含 actions）
		doneData, _ := json.Marshal(map[string]interface{}{
			"type":    "done",
			"sources": resp.Sources,
			"actions": resp.Actions,
		})
		c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", string(doneData))))
		flusher.Flush()
	}()

	for chunk := range chunkChan {
		chunkData, _ := json.Marshal(map[string]interface{}{
			"type":    "chunk",
			"content": chunk,
		})
		c.Writer.Write([]byte(fmt.Sprintf("data: %s\n\n", string(chunkData))))
		flusher.Flush()
	}
}

// ClearSession 清空会话
func (h *ChatHandler) ClearSession(c *gin.Context) {
	ctx := context.Background()
	sessionID := c.Query("session_id")

	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
		return
	}

	if err := h.Session.Clear(ctx, sessionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "cleared"})
}
