package models

import "time"

// RefundRequest 退款请求
type RefundRequest struct {
	OrderID      string    `json:"order_id"`
	UserID       string    `json:"user_id"`
	Reason       string    `json:"reason"`
	RefundAmount float64   `json:"refund_amount"`
	Status       string    `json:"status"` // pending, approved, rejected, completed
	CreatedAt    time.Time `json:"created_at"`
}

// RefundResult 退款结果
type RefundResult struct {
	Success      bool    `json:"success"`
	Message      string  `json:"message"`
	RefundID     string  `json:"refund_id"`
	OrderID      string  `json:"order_id"`
	Status       string  `json:"status"`
	RefundAmount float64 `json:"refund_amount"`
}
