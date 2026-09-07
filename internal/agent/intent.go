package agent

import (
	"encoding/json"
	"fmt"
	"strings"

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

// Analyze 分析用户意图（无上下文）
func (ia *IntentAnalyzer) Analyze(userMessage string) (*models.IntentResult, error) {
	return ia.AnalyzeWithContext(userMessage, []models.Message{})
}

// AnalyzeWithContext 基于上下文分析意图
func (ia *IntentAnalyzer) AnalyzeWithContext(userMessage string, history []models.Message) (*models.IntentResult, error) {
	// 构建上下文信息
	var contextStr string
	if len(history) > 0 {
		contextStr = "历史对话:\n"
		// 只取最近6条
		start := 0
		if len(history) > 6 {
			start = len(history) - 6
		}
		for _, msg := range history[start:] {
			role := "用户"
			if msg.Role == "assistant" {
				role = "助手"
			}
			// 截断过长的消息
			content := msg.Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}
			contextStr += role + ": " + content + "\n"
		}
		contextStr += "\n"
	}

	systemPrompt := `你是一个电商客服意图识别专家。分析用户的问题，识别其意图并提取关键实体。

意图分类：
- order_query: 查询订单状态、物流信息
- return_refund: 退货、退款申请
- payment_issue: 支付问题、支付失败
- account_issue: 账户问题、密码重置
- shipping_issue: 配送问题、运费咨询
- quality_issue: 商品质量问题、售后
- coupon_issue: 优惠券查询、使用
- account_cancel: 注销账户
- general_inquiry: 一般咨询（无法归类的其他问题）

实体提取规则（重要）：
1. 订单号：如果用户提到了订单号（格式如 ORD-xxx、#xxx、纯数字6-12位），提取为 order_id
   - 例如："ORD-20260907-001" → "order_id": "ORD-20260907-001"
   - 例如："订单号123456789" → "order_id": "123456789"
   - 例如："#987654" → "order_id": "987654"

2. 商品名称：如果用户提到了具体商品，提取为 product_name
   - 例如："我买的耳机" → "product_name": "耳机"
   - 例如："iPhone 15" → "product_name": "iPhone 15"

3. 金额：如果用户提到了金额，提取为 amount
   - 例如："300块钱" → "amount": "300"
   - 例如："花了299元" → "amount": "299"

4. 日期：如果用户提到了时间，提取为 date
   - 例如："昨天的订单" → "date": "昨天"
   - 例如："9月5号的" → "date": "9月5号"

5. 时间范围：如果用户提到"最近"、"最新"、"上一笔"，提取为 time_range
   - 例如："最近一笔" → "time_range": "recent"
   - 例如："最新订单" → "time_range": "latest"

6. 物流相关：如果用户提到"物流"、"快递"、"到哪了"，标记 logistics_query
   - 例如："快递到哪了" → "logistics_query": "true"

7. 退货原因：如果用户提到退货，提取退货原因
   - 例如："质量不好想退货" → "return_reason": "质量问题"

重要：
1. 结合历史对话上下文，理解用户的指代（如"这个订单"、"刚才说的那个"、"它"等代词）
2. 如果历史中提到了订单号、商品等信息，自动继承到当前实体的entities中
3. 如果用户的问题与历史相关，confidence可以适当提高
4. 如果用户明确提到了订单号，confidence应该提高

返回JSON格式：
{
  "intent": "意图分类",
  "confidence": 0.0-1.0之间的置信度,
  "entities": {
    "order_id": "订单号（如果有）",
    "product_name": "商品名称（如果有）",
    "amount": "金额（如果有）",
    "date": "日期（如果有）",
    "time_range": "时间范围（如果有）",
    "logistics_query": "true/false",
    "return_reason": "退货原因（如果有）"
  },
  "need_transfer": false,
  "reason": "如果需要转人工，说明原因"
}

只返回JSON，不要有其他内容。`

	userPrompt := fmt.Sprintf("%s当前用户问题: %s", contextStr, userMessage)

	reply, err := ia.LLMClient.ChatWithSystem(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM调用失败: %w", err)
	}

	// 清理可能的markdown标记
	reply = strings.TrimSpace(reply)
	reply = strings.TrimPrefix(reply, "```json")
	reply = strings.TrimPrefix(reply, "```")
	reply = strings.TrimSuffix(reply, "```")

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

	// 确保Entities不为nil
	if result.Entities == nil {
		result.Entities = map[string]string{}
	}

	return &result, nil
}
