package agent

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/LittleCurry/go_first_ai/internal/config"
	"github.com/LittleCurry/go_first_ai/internal/llm"
	"github.com/LittleCurry/go_first_ai/internal/models"
	"github.com/LittleCurry/go_first_ai/internal/rag"
	"github.com/LittleCurry/go_first_ai/internal/tools"
)

// Agent 对话代理
type Agent struct {
	LLMClient      *llm.Client
	Retriever      *rag.Retriever
	IntentAnalyzer *IntentAnalyzer
	OrderTool      *tools.OrderTool
	MaxIterations  int
}

// NewAgent 创建Agent
func NewAgent(cfg *config.Config) *Agent {
	llmClient := llm.NewClient(cfg)
	return &Agent{
		LLMClient:      llmClient,
		Retriever:      rag.NewRetriever(cfg),
		IntentAnalyzer: NewIntentAnalyzer(llmClient),
		OrderTool:      tools.NewOrderTool(),
		MaxIterations:  cfg.Agent.MaxIterations,
	}
}

// ProcessStream 流式处理用户消息
func (a *Agent) ProcessStream(ctx context.Context, userMessage string, history []models.Message, chunkChan chan<- string) (*AgentResponse, error) {
	// 检测是否是 Action 触发的消息
	if strings.HasPrefix(userMessage, "/action:") {
		return a.handleAction(ctx, userMessage, history, chunkChan)
	}

	// Step 1: 意图识别
	intentResult, err := a.IntentAnalyzer.AnalyzeWithContext(userMessage, history)
	if err != nil {
		return nil, fmt.Errorf("意图识别失败: %w", err)
	}

	// Step 2: 检查是否需要转人工
	if intentResult.Confidence < 0.3 || intentResult.NeedTransfer {
		return &AgentResponse{
			Reply:        "抱歉，您的问题比较复杂，我帮您转接人工客服。",
			Intent:       intentResult.Intent,
			Confidence:   intentResult.Confidence,
			NeedTransfer: true,
			Reason:       intentResult.Reason,
		}, nil
	}

	// Step 3: 根据意图路由
	switch intentResult.Intent {
	case "order_query":
		return a.handleOrderQuery(ctx, userMessage, intentResult, history, chunkChan)
	case "coupon_issue":
		return a.handleCouponQuery(ctx, userMessage, intentResult, history, chunkChan)
	case "return_refund":
		return a.handleRefund(ctx, userMessage, intentResult, history, chunkChan)
	default:
		return a.handleRAGQuery(ctx, userMessage, intentResult, history, chunkChan)
	}
}

// handleAction 处理Action触发的消息
func (a *Agent) handleAction(ctx context.Context, userMessage string, history []models.Message, chunkChan chan<- string) (*AgentResponse, error) {
	// 解析 /action:confirm_refund|order_id=xxx|reason=xxx
	parts := strings.SplitN(strings.TrimPrefix(userMessage, "/action:"), "|", 2)
	action := parts[0]

	entities := map[string]string{}
	if len(parts) > 1 {
		for _, pair := range strings.Split(parts[1], "|") {
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				entities[kv[0]] = kv[1]
			}
		}
	}

	intent := &models.IntentResult{
		Entities: entities,
	}

	switch action {
	case "confirm_refund":
		return a.handleConfirmRefund(ctx, "确认退款", intent, history, chunkChan)
	case "track_logistics":
		return a.handleLogistics(ctx, "查询物流", intent, history, chunkChan)
	case "repurchase":
		return a.handleRepurchase(ctx, "再来一单", intent, history, chunkChan)
	case "view_order":
		return a.handleOrderQuery(ctx, "查看订单", intent, history, chunkChan)
	case "view_coupons":
		return a.handleCouponQuery(ctx, "查看优惠券", intent, history, chunkChan)
	default:
		return a.handleRAGQuery(ctx, action, intent, history, chunkChan)
	}
}

// handleRAGQuery 处理普通RAG查询
func (a *Agent) handleRAGQuery(ctx context.Context, userMessage string, intent *models.IntentResult, history []models.Message, chunkChan chan<- string) (*AgentResponse, error) {
	context, ragResults, err := a.Retriever.RetrieveForChat(userMessage)
	if err != nil {
		return nil, fmt.Errorf("RAG检索失败: %w", err)
	}

	reply, err := a.generateReplyStream(ctx, userMessage, context, intent, history, chunkChan)
	if err != nil {
		return nil, err
	}

	sources := make([]string, 0, len(ragResults))
	for _, r := range ragResults {
		if r.Source != "" {
			sources = append(sources, r.Source)
		}
	}

	return &AgentResponse{
		Reply:      reply,
		Intent:     intent.Intent,
		Confidence: intent.Confidence,
		Sources:    sources,
	}, nil
}

// generateReplyStream 流式生成回复
func (a *Agent) generateReplyStream(ctx context.Context, userMessage, context string, intent *models.IntentResult, history []models.Message, chunkChan chan<- string) (string, error) {
	systemPrompt := `你是一个专业、友好的电商客服AI助手。

核心原则：
1. 基于提供的知识库内容回答问题，不要编造信息
2. 如果知识库中没有相关信息，如实告知并建议转人工
3. 回复要简洁、清晰、有温度
4. 使用礼貌用语，称呼用户为"您"
5. 如果用户问的是关于之前对话的问题，一定要结合历史对话来回答`

	var contextBlock string
	if context != "" {
		contextBlock = fmt.Sprintf("知识库参考:\n%s", context)
	} else {
		contextBlock = "知识库中未找到相关信息，请告知用户暂时无法回答，建议转人工。"
	}

	var historyBlock string
	if len(history) > 0 {
		historyBlock = "历史对话:\n"
		for _, msg := range history {
			role := "用户"
			if msg.Role == "assistant" {
				role = "助手"
			}
			historyBlock += fmt.Sprintf("%s: %s\n", role, msg.Content)
		}
		historyBlock += "\n"
	}

	userPrompt := fmt.Sprintf(`%s当前用户问题: %s

意图识别结果: %s (置信度: %.2f)

%s

请基于以上信息生成回复。`,
		historyBlock,
		userMessage,
		intent.Intent,
		intent.Confidence,
		contextBlock,
	)

	return a.LLMClient.ChatWithSystemStream(ctx, systemPrompt, userPrompt, chunkChan)
}

// generateReplyWithData 基于实时数据生成回复
func (a *Agent) generateReplyWithData(ctx context.Context, userMessage, dataContext string, intent *models.IntentResult, history []models.Message, chunkChan chan<- string) (string, error) {
	systemPrompt := `你是一个专业、友好的电商客服AI助手。

核心原则：
1. 基于提供的实时数据回答问题，这些数据是真实的业务数据
2. 将数据以清晰、友好的方式呈现给用户
3. 可以适当添加emoji增加亲和力
4. 回复要简洁、有温度`

	var historyBlock string
	if len(history) > 0 {
		historyBlock = "历史对话:\n"
		for _, msg := range history {
			role := "用户"
			if msg.Role == "assistant" {
				role = "助手"
			}
			historyBlock += fmt.Sprintf("%s: %s\n", role, msg.Content)
		}
		historyBlock += "\n"
	}

	userPrompt := fmt.Sprintf(`%s用户问题: %s
意图: %s

实时数据:
%s

请基于以上数据，生成专业、友好的回复。`,
		historyBlock,
		userMessage,
		intent.Intent,
		dataContext,
	)

	return a.LLMClient.ChatWithSystemStream(ctx, systemPrompt, userPrompt, chunkChan)
}

// AgentResponse 扩展，增加 Actions
type AgentResponse struct {
	Reply        string
	Intent       string
	Confidence   float64
	Sources      []string
	NeedTransfer bool
	Reason       string
	Data         interface{}
	Actions      []models.Action // 新增
}

// handleCouponQuery 修改为返回 Actions
func (a *Agent) handleCouponQuery(ctx context.Context, userMessage string, intent *models.IntentResult, history []models.Message, chunkChan chan<- string) (*AgentResponse, error) {
	userID := "333"
	coupons, err := a.OrderTool.GetUserCoupons(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("查询优惠券失败: %w", err)
	}

	if len(coupons) == 0 {
		chunkChan <- "💳 您目前没有可用的优惠券。"
		return &AgentResponse{
			Reply:      "您目前没有可用的优惠券，可以关注我们的活动获取更多优惠。",
			Intent:     intent.Intent,
			Confidence: intent.Confidence,
			Actions: []models.Action{
				{Label: "🛍️ 去购物", Action: "browse_shop", Data: map[string]interface{}{}},
			},
		}, nil
	}

	// 格式化优惠券信息
	var couponContext strings.Builder
	couponContext.WriteString("【您的优惠券】\n")
	for _, c := range coupons {
		discountText := ""
		if c.DiscountType == "percentage" {
			discountText = fmt.Sprintf("%.0f折", 10-c.DiscountValue/10)
		} else {
			discountText = fmt.Sprintf("%.2f元", c.DiscountValue)
		}
		couponContext.WriteString(fmt.Sprintf("  • %s (%s", c.Name, discountText))
		if c.MinAmount > 0 {
			couponContext.WriteString(fmt.Sprintf(", 满%.2f可用", c.MinAmount))
		}
		couponContext.WriteString(fmt.Sprintf(") 有效期至：%s\n", c.ExpireAt.Format("2006-01-02")))
	}

	chunkChan <- "💳 正在为您查询优惠券...\n"

	reply, err := a.generateReplyWithData(ctx, userMessage, couponContext.String(), intent, history, chunkChan)
	if err != nil {
		log.Printf("生成回复失败: %v", err)
	}

	return &AgentResponse{
		Reply:      reply,
		Intent:     intent.Intent,
		Confidence: intent.Confidence,
		Sources:    []string{"实时优惠券数据"},
		Data:       coupons,
		Actions: []models.Action{
			{Label: "🛍️ 去购物", Action: "browse_shop", Data: map[string]interface{}{}},
			{Label: "📋 查看订单", Action: "view_orders", Data: map[string]interface{}{}},
		},
	}, nil
}

// handleLogistics 处理物流查询
func (a *Agent) handleLogistics(ctx context.Context, userMessage string, intent *models.IntentResult, history []models.Message, chunkChan chan<- string) (*AgentResponse, error) {
	userID := "333"

	var order *models.Order
	var err error

	if orderID, ok := intent.Entities["order_id"]; ok && orderID != "" {
		order, err = a.OrderTool.GetOrderByID(ctx, userID, orderID)
	} else {
		order, err = a.OrderTool.GetRecentOrder(ctx, userID)
	}

	if err != nil || order == nil {
		chunkChan <- "🚚 未查询到物流信息。"
		return &AgentResponse{
			Reply:      "未查询到物流信息，请确认订单状态或联系人工客服。",
			Intent:     intent.Intent,
			Confidence: intent.Confidence,
		}, nil
	}

	if order.TrackingNumber == "" {
		chunkChan <- "📦 该订单尚未发货，暂无物流信息。"
		return &AgentResponse{
			Reply:      fmt.Sprintf("订单 %s 尚未发货，暂无物流信息。请耐心等待商家发货。", order.OrderID),
			Intent:     intent.Intent,
			Confidence: intent.Confidence,
			Actions: []models.Action{
				{Label: "📋 查看订单", Action: "view_order", Data: map[string]interface{}{
					"order_id": order.OrderID,
				}},
			},
		}, nil
	}

	logistics, err := a.OrderTool.GetOrderLogistics(ctx, order.OrderID)
	if err != nil || len(logistics) == 0 {
		chunkChan <- "🚚 暂未查询到物流轨迹。"
		return &AgentResponse{
			Reply:      "暂未查询到物流轨迹，请稍后重试。",
			Intent:     intent.Intent,
			Confidence: intent.Confidence,
		}, nil
	}

	var logisticsText strings.Builder
	logisticsText.WriteString(fmt.Sprintf("🚚 订单 %s 物流信息：\n", order.OrderID))
	logisticsText.WriteString(fmt.Sprintf("物流公司：%s %s\n\n", order.TrackingCompany, order.TrackingNumber))
	for _, l := range logistics {
		logisticsText.WriteString(fmt.Sprintf("  • %s %s\n", l.Time.Format("01-02 15:04"), l.Description))
	}

	chunkChan <- logisticsText.String()

	return &AgentResponse{
		Reply:      logisticsText.String(),
		Intent:     intent.Intent,
		Confidence: intent.Confidence,
		Actions: []models.Action{
			{Label: "🔄 刷新物流", Action: "refresh_logistics", Data: map[string]interface{}{
				"order_id": order.OrderID,
			}},
			{Label: "📋 查看订单", Action: "view_order", Data: map[string]interface{}{
				"order_id": order.OrderID,
			}},
		},
	}, nil
}

// handleRepurchase 处理再来一单
func (a *Agent) handleRepurchase(ctx context.Context, userMessage string, intent *models.IntentResult, history []models.Message, chunkChan chan<- string) (*AgentResponse, error) {
	userID := "333"

	var order *models.Order
	var err error

	if orderID, ok := intent.Entities["order_id"]; ok && orderID != "" {
		order, err = a.OrderTool.GetOrderByID(ctx, userID, orderID)
	} else {
		order, err = a.OrderTool.GetRecentOrder(ctx, userID)
	}

	if err != nil || order == nil {
		chunkChan <- "🔄 未找到可重新购买的订单。"
		return &AgentResponse{
			Reply:      "未找到可重新购买的订单。",
			Intent:     intent.Intent,
			Confidence: intent.Confidence,
		}, nil
	}

	// 创建新订单
	newOrder, err := a.OrderTool.RepurchaseOrder(ctx, userID, order.OrderID)
	if err != nil {
		log.Printf("再来一单失败: %v", err)
		chunkChan <- "❌ 再来一单失败，请稍后重试。"
		return &AgentResponse{
			Reply:        "再来一单失败，请稍后重试或联系人工客服。",
			Intent:       intent.Intent,
			Confidence:   intent.Confidence,
			NeedTransfer: true,
		}, nil
	}

	orderContext := tools.FormatOrderForDisplay(newOrder)
	chunkChan <- "🔄 正在为您重新下单...\n"
	chunkChan <- orderContext + "\n"

	reply := fmt.Sprintf("✅ 已为您重新下单！\n新订单号：%s\n金额：%.2f元\n状态：待发货", newOrder.OrderID, newOrder.TotalAmount)

	return &AgentResponse{
		Reply:      reply,
		Intent:     intent.Intent,
		Confidence: intent.Confidence,
		Actions: []models.Action{
			{Label: "🚚 查看物流", Action: "track_logistics", Data: map[string]interface{}{
				"order_id": newOrder.OrderID,
			}},
			{Label: "📋 查看订单", Action: "view_order", Data: map[string]interface{}{
				"order_id": newOrder.OrderID,
			}},
		},
	}, nil
}

// handleConfirmRefund 处理确认退款
func (a *Agent) handleConfirmRefund(ctx context.Context, userMessage string, intent *models.IntentResult, history []models.Message, chunkChan chan<- string) (*AgentResponse, error) {
	userID := "333"

	// 从entities中提取订单号
	orderID, ok := intent.Entities["order_id"]
	if !ok || orderID == "" {
		chunkChan <- "❌ 未找到订单信息，请重新输入。"
		return &AgentResponse{
			Reply:      "未找到订单信息，请重新输入。",
			Intent:     intent.Intent,
			Confidence: intent.Confidence,
		}, nil
	}

	reason, _ := intent.Entities["return_reason"]
	if reason == "" {
		reason = "用户申请退款"
	}

	// 执行退款
	result, err := a.OrderTool.ProcessRefund(ctx, userID, orderID, reason)
	if err != nil {
		log.Printf("退款失败: %v", err)
		chunkChan <- "❌ 退款处理失败，请稍后重试。"
		return &AgentResponse{
			Reply:        "退款处理失败，请稍后重试或联系人工客服。",
			Intent:       intent.Intent,
			Confidence:   intent.Confidence,
			NeedTransfer: true,
		}, nil
	}

	if !result.Success {
		chunkChan <- result.Message
		return &AgentResponse{
			Reply:      result.Message,
			Intent:     intent.Intent,
			Confidence: intent.Confidence,
			Actions: []models.Action{
				{Label: "📞 转人工客服", Action: "transfer_human", Data: map[string]interface{}{}},
			},
		}, nil
	}

	chunkChan <- "💰 正在处理退款申请...\n"
	chunkChan <- result.Message

	return &AgentResponse{
		Reply:      result.Message,
		Intent:     intent.Intent,
		Confidence: intent.Confidence,
		Actions: []models.Action{
			{Label: "📋 查看退款进度", Action: "view_refund", Data: map[string]interface{}{
				"refund_id": result.RefundID,
				"order_id":  result.OrderID,
			}},
			{Label: "📞 联系人工", Action: "transfer_human", Data: map[string]interface{}{}},
		},
	}, nil
}

// handleRefund 处理退款
func (a *Agent) handleRefund(ctx context.Context, userMessage string, intent *models.IntentResult, history []models.Message, chunkChan chan<- string) (*AgentResponse, error) {
	userID := "333"

	// 先检查用户是否有订单
	order, err := a.OrderTool.GetRecentOrder(ctx, userID)
	if err != nil {
		log.Printf("查询订单失败: %v", err)
		chunkChan <- "❌ 查询订单失败，请稍后重试。"
		return &AgentResponse{
			Reply:        "查询订单失败，请稍后重试或联系人工客服。",
			Intent:       intent.Intent,
			Confidence:   intent.Confidence,
			NeedTransfer: true,
		}, nil
	}

	if order == nil || order.OrderID == "" {
		log.Printf("用户 %s 没有订单", userID)
		chunkChan <- "📭 您目前没有可退款的订单。"
		return &AgentResponse{
			Reply:      "您目前没有可退款的订单。如需帮助，请联系人工客服。",
			Intent:     intent.Intent,
			Confidence: intent.Confidence,
			Actions: []models.Action{
				{Label: "🛍️ 去购物", Action: "browse_shop", Data: map[string]interface{}{}},
				{Label: "📞 转人工客服", Action: "transfer_human", Data: map[string]interface{}{}},
			},
		}, nil
	}

	// 检查是否可退款
	allowedStatus := map[string]bool{
		"paid":    true,
		"shipped": true,
	}
	if !allowedStatus[order.Status] {
		chunkChan <- fmt.Sprintf("⚠️ 订单 %s 当前状态为【%s】，无法申请退款。", order.OrderID, order.StatusText)
		return &AgentResponse{
			Reply:      fmt.Sprintf("订单 %s 当前状态为【%s】，无法申请退款。如需帮助，请联系人工客服。", order.OrderID, order.StatusText),
			Intent:     intent.Intent,
			Confidence: intent.Confidence,
			Actions: []models.Action{
				{Label: "📞 转人工客服", Action: "transfer_human", Data: map[string]interface{}{}},
			},
		}, nil
	}

	// 展示退款确认
	orderContext := tools.FormatOrderForDisplay(order)
	chunkChan <- "💰 正在为您查询退款信息...\n"
	chunkChan <- orderContext + "\n"

	reply := fmt.Sprintf(`请确认退款信息：

📋 订单号：%s
💰 退款金额：%.2f元
📦 订单状态：%s

确认无误后点击下方"确认退款"按钮，系统将为您处理。`,
		order.OrderID,
		order.TotalAmount,
		order.StatusText,
	)

	chunkChan <- reply

	return &AgentResponse{
		Reply:      reply,
		Intent:     intent.Intent,
		Confidence: intent.Confidence,
		Actions: []models.Action{
			{Label: "✅ 确认退款", Action: "confirm_refund", Data: map[string]interface{}{
				"order_id": order.OrderID,
				"reason":   intent.Entities["return_reason"],
			}},
			{Label: "📞 转人工客服", Action: "transfer_human", Data: map[string]interface{}{}},
			{Label: "📋 查看订单详情", Action: "view_order", Data: map[string]interface{}{
				"order_id": order.OrderID,
			}},
		},
	}, nil
}

// handleOrderQuery 处理订单查询 - 确保返回 Actions
func (a *Agent) handleOrderQuery(ctx context.Context, userMessage string, intent *models.IntentResult, history []models.Message, chunkChan chan<- string) (*AgentResponse, error) {
	userID := "333"

	var order *models.Order
	var err error

	if orderID, ok := intent.Entities["order_id"]; ok && orderID != "" {
		order, err = a.OrderTool.GetOrderByID(ctx, userID, orderID)
	} else {
		order, err = a.OrderTool.GetRecentOrder(ctx, userID)
	}

	if err != nil {
		log.Printf("查询订单失败: %v", err)
		chunkChan <- "❌ 查询订单时遇到问题，请稍后重试。"
		return &AgentResponse{
			Reply:        "查询订单时遇到技术问题，建议您稍后重试或联系人工客服。",
			Intent:       intent.Intent,
			Confidence:   intent.Confidence,
			NeedTransfer: true,
			Reason:       "数据库查询失败",
			Actions: []models.Action{
				{Label: "📞 转人工客服", Action: "transfer_human", Data: map[string]interface{}{}},
			},
		}, nil
	}

	if order == nil || order.OrderID == "" {
		chunkChan <- "📭 您目前没有订单记录。"
		return &AgentResponse{
			Reply:      "您目前没有订单记录，如需购物可浏览我们的商品页面。",
			Intent:     intent.Intent,
			Confidence: intent.Confidence,
			Actions: []models.Action{
				{Label: "🛍️ 去购物", Action: "browse_shop", Data: map[string]interface{}{}},
			},
		}, nil
	}

	// 格式化订单信息
	orderContext := tools.FormatOrderForDisplay(order)
	chunkChan <- "📦 正在为您查询订单信息...\n"
	chunkChan <- orderContext + "\n"

	// 生成回复
	reply, err := a.generateReplyWithData(ctx, userMessage, orderContext, intent, history, chunkChan)
	if err != nil {
		log.Printf("生成回复失败: %v", err)
		chunkChan <- "📦 订单查询结果如上。"
	}

	// 生成 Actions
	actions := a.generateOrderActions(order)

	return &AgentResponse{
		Reply:      reply,
		Intent:     intent.Intent,
		Confidence: intent.Confidence,
		Sources:    []string{"实时订单数据"},
		Data:       order,
		Actions:    actions,
	}, nil
}

// generateOrderActions 生成订单相关的 Actions
func (a *Agent) generateOrderActions(order *models.Order) []models.Action {
	var actions []models.Action

	// 1. 再来一单
	actions = append(actions, models.Action{
		Label:  "🔄 再来一单",
		Action: "repurchase",
		Data: map[string]interface{}{
			"order_id": order.OrderID,
		},
	})

	// 2. 查看物流（如果已发货）
	if order.Status == "shipped" || order.Status == "delivered" {
		actions = append(actions, models.Action{
			Label:  "🚚 查看物流",
			Action: "track_logistics",
			Data: map[string]interface{}{
				"order_id": order.OrderID,
			},
		})
	}

	// 3. 申请退款（如果可退款）
	if order.Status == "paid" || order.Status == "shipped" {
		actions = append(actions, models.Action{
			Label:  "💰 申请退款",
			Action: "refund",
			Data: map[string]interface{}{
				"order_id": order.OrderID,
			},
		})
	}

	// 4. 查看优惠券
	actions = append(actions, models.Action{
		Label:  "🎫 查看优惠券",
		Action: "view_coupons",
		Data:   map[string]interface{}{},
	})

	// 5. 转人工
	actions = append(actions, models.Action{
		Label:  "💬 转人工客服",
		Action: "transfer_human",
		Data:   map[string]interface{}{},
	})

	return actions
}
