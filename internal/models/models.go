package models

import (
	"time"
)

// ChatSession 对话会话
type ChatSession struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"` // active, resolved, transferred
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Message 单条消息
type Message struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	Role       string    `json:"role"` // user, assistant, system
	Content    string    `json:"content"`
	Intent     string    `json:"intent,omitempty"`     // 意图识别结果
	Entities   string    `json:"entities,omitempty"`   // 抽取的实体（JSON）
	Confidence float64   `json:"confidence,omitempty"` // 置信度
	Sources    []string  `json:"sources,omitempty"`    // RAG引用的知识来源
	CreatedAt  time.Time `json:"created_at"`
}

// ChatRequest API请求
type ChatRequest struct {
	SessionID string    `json:"session_id,omitempty"` // 空则创建新会话
	UserID    string    `json:"user_id"`
	Message   string    `json:"message"`
	History   []Message `json:"history,omitempty"` // 最近N轮对话
}

// ChatResponse API响应
type ChatResponse struct {
	SessionID      string   `json:"session_id"`
	MessageID      string   `json:"message_id"`
	Reply          string   `json:"reply"`
	Intent         string   `json:"intent"`
	Confidence     float64  `json:"confidence"`
	Sources        []string `json:"sources,omitempty"`
	NeedTransfer   bool     `json:"need_transfer,omitempty"` // 是否需要转人工
	TransferReason string   `json:"transfer_reason,omitempty"`
}

// KnowledgeEntry 知识库条目
type KnowledgeEntry struct {
	ID               string    `json:"id"`
	Category         string    `json:"category"`          // 分类：订单/退货/支付/账户等
	Question         string    `json:"question"`          // 标准问题
	Answer           string    `json:"answer"`            // 标准答案
	Keywords         []string  `json:"keywords"`          // 关键词
	SimilarQuestions []string  `json:"similar_questions"` // 相似问法
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// IntentResult 意图识别结果
type IntentResult struct {
	Intent       string            `json:"intent"`
	Confidence   float64           `json:"confidence"`
	Entities     map[string]string `json:"entities"` // 如 {"order_id": "123456"}
	NeedTransfer bool              `json:"need_transfer"`
	Reason       string            `json:"reason,omitempty"`
}

// RAGResult RAG检索结果
type RAGResult struct {
	Content  string  `json:"content"`
	Score    float64 `json:"score"`
	Source   string  `json:"source"` // 知识ID或文档名
	Category string  `json:"category"`
}
