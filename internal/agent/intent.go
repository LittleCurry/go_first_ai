package agent

import (
	"encoding/json"
	"fmt"

	"github.com/LittleCurry/go_first_ai/internal/llm"
	"github.com/LittleCurry/go_first_ai/internal/models"
)

// IntentAnalyzer 意图分析器
type IntentAnalyzer struct {
	LLMClient *llm.Client
}

// NewIntentAnalyzer 创建意图分析器
func NewIntentAnalyzer(client *llm.Client) *IntentAnalyzer {
	return &IntentAnalyzer{
		LLMClient: client,
	}
}

// Analyze 分析用户意图
func (ia *IntentAnalyzer) Analyze(userMessage string) (*models.IntentResult, error) {
	// 构建Prompt
	systemPrompt := `你是一个电商客服意图识别专家。分析用户的问题，识别其意图并提取关键实体。

意图分类：
- order_query: 查询订单状态、物流信息
- return_refund: 退货、退款申请
- payment_issue: 支付问题、支付失败
- account_issue: 账户问题、密码重置
- shipping_issue: 配送问题、运费咨询
- quality_issue: 商品质量问题、售后
- coupon_issue: 优惠券使用
- account_cancel: 注销账户
- general_inquiry: 一般咨询（无法归类的其他问题）

你需要返回JSON格式：
{
  "intent": "意图分类",
  "confidence": 0.0-1.0之间的置信度,
  "entities": {"key": "value"},
  "need_transfer": false,
  "reason": "如果需要转人工，说明原因"
}

实体提取规则：
- 如果有订单号，提取为 order_id
- 如果有商品名称，提取为 product_name
- 如果有金额，提取为 amount
- 如果有日期，提取为 date

只返回JSON，不要有其他内容。`

	userPrompt := fmt.Sprintf("用户问题: %s", userMessage)

	reply, err := ia.LLMClient.ChatWithSystem(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM调用失败: %w", err)
	}

	// 解析JSON
	var result models.IntentResult
	if err := json.Unmarshal([]byte(reply), &result); err != nil {
		// 如果解析失败，返回默认意图
		return &models.IntentResult{
			Intent:       "general_inquiry",
			Confidence:   0.5,
			Entities:     map[string]string{},
			NeedTransfer: false,
		}, nil
	}

	return &result, nil
}
