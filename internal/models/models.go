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

// Action 建议操作按钮
type Action struct {
	Label  string                 `json:"label"`  // 按钮显示文字
	Action string                 `json:"action"` // 操作类型: repurchase, refund, track, view_order, view_coupons
	Data   map[string]interface{} `json:"data"`   // 操作参数
}

// ChatResponse API响应
type ChatResponse struct {
	SessionID      string   `json:"session_id"`
	MessageID      string   `json:"message_id"`
	Reply          string   `json:"reply"`
	Intent         string   `json:"intent"`
	Confidence     float64  `json:"confidence"`
	Sources        []string `json:"sources,omitempty"`
	NeedTransfer   bool     `json:"need_transfer,omitempty"`
	TransferReason string   `json:"transfer_reason,omitempty"`
	Actions        []Action `json:"actions,omitempty"` // 新增：建议操作按钮
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

// Order 订单
type Order struct {
	OrderID         string      `json:"order_id"`
	UserID          string      `json:"user_id"`
	Status          string      `json:"status"`
	StatusText      string      `json:"status_text"`
	TotalAmount     float64     `json:"total_amount"`
	ShippingAddress string      `json:"shipping_address"`
	TrackingNumber  string      `json:"tracking_number"`
	TrackingCompany string      `json:"tracking_company"`
	Items           []OrderItem `json:"items"`
	Logistics       []Logistics `json:"logistics"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// OrderItem 订单商品
type OrderItem struct {
	ProductName  string  `json:"product_name"`
	ProductImage string  `json:"product_image"`
	Quantity     int     `json:"quantity"`
	Price        float64 `json:"price"`
	Subtotal     float64 `json:"subtotal"`
}

// Logistics 物流轨迹
type Logistics struct {
	Status      string    `json:"status"`
	StatusText  string    `json:"status_text"`
	Location    string    `json:"location"`
	Description string    `json:"description"`
	Time        time.Time `json:"time"`
}

// Coupon 优惠券
type Coupon struct {
	CouponCode    string    `json:"coupon_code"`
	Name          string    `json:"name"`
	DiscountType  string    `json:"discount_type"`
	DiscountValue float64   `json:"discount_value"`
	MinAmount     float64   `json:"min_amount"`
	Status        string    `json:"status"`
	ExpireAt      time.Time `json:"expire_at"`
}
