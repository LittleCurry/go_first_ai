package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/LittleCurry/go_first_ai/internal/config"
	"github.com/LittleCurry/go_first_ai/internal/llm"
	"github.com/LittleCurry/go_first_ai/internal/models"
	"github.com/LittleCurry/go_first_ai/internal/rag"
)

// Agent 对话代理
type Agent struct {
	LLMClient      *llm.Client
	Retriever      *rag.Retriever
	IntentAnalyzer *IntentAnalyzer
	MaxIterations  int
}

// NewAgent 创建Agent
func NewAgent(cfg *config.Config) *Agent {
	llmClient := llm.NewClient(cfg)
	return &Agent{
		LLMClient:      llmClient,
		Retriever:      rag.NewRetriever(cfg),
		IntentAnalyzer: NewIntentAnalyzer(llmClient),
		MaxIterations:  cfg.Agent.MaxIterations,
	}
}

// AgentResponse Agent响应
type AgentResponse struct {
	Reply        string
	Intent       string
	Confidence   float64
	Sources      []string
	NeedTransfer bool
	Reason       string
}

// Process 处理用户消息
func (a *Agent) Process(userMessage string, history []models.Message) (*AgentResponse, error) {
	// Step 1: 意图识别
	intentResult, err := a.IntentAnalyzer.Analyze(userMessage)
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

	// Step 3: RAG检索
	context, ragResults, err := a.Retriever.RetrieveForChat(userMessage)
	if err != nil {
		return nil, fmt.Errorf("RAG检索失败: %w", err)
	}

	// Step 4: 生成回答
	reply, err := a.generateReply(userMessage, context, intentResult, history)
	if err != nil {
		return nil, fmt.Errorf("生成回答失败: %w", err)
	}

	// Step 5: 收集来源
	sources := make([]string, 0, len(ragResults))
	for _, r := range ragResults {
		if r.Source != "" {
			sources = append(sources, r.Source)
		}
	}

	return &AgentResponse{
		Reply:        reply,
		Intent:       intentResult.Intent,
		Confidence:   intentResult.Confidence,
		Sources:      sources,
		NeedTransfer: false,
	}, nil
}

// ProcessStream 流式处理用户消息（通过channel发送片段）
func (a *Agent) ProcessStream(ctx context.Context, userMessage string, history []models.Message, chunkChan chan<- string) (*AgentResponse, error) {
	// Step 1: 意图识别
	intentResult, err := a.IntentAnalyzer.Analyze(userMessage)
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

	// Step 3: RAG检索
	context, ragResults, err := a.Retriever.RetrieveForChat(userMessage)
	if err != nil {
		return nil, fmt.Errorf("RAG检索失败: %w", err)
	}

	// Step 4: 流式生成回答
	fullReply, err := a.generateReplyStream(ctx, userMessage, context, intentResult, history, chunkChan)
	if err != nil {
		return nil, fmt.Errorf("生成回答失败: %w", err)
	}

	// Step 5: 收集来源
	sources := make([]string, 0, len(ragResults))
	for _, r := range ragResults {
		if r.Source != "" {
			sources = append(sources, r.Source)
		}
	}

	return &AgentResponse{
		Reply:        fullReply,
		Intent:       intentResult.Intent,
		Confidence:   intentResult.Confidence,
		Sources:      sources,
		NeedTransfer: false,
	}, nil
}

// generateReplyStream 流式生成回复
func (a *Agent) generateReplyStream(ctx context.Context, userMessage, context string, intent *models.IntentResult, history []models.Message, chunkChan chan<- string) (string, error) {
	// 构建系统提示
	systemPrompt := `你是一个专业、友好的电商客服AI助手。

核心原则：
1. 基于提供的知识库内容回答问题，不要编造信息
2. 如果知识库中没有相关信息，如实告知并建议转人工
3. 回复要简洁、清晰、有温度
4. 使用礼貌用语，称呼用户为"您"

回复格式要求：
- 使用简洁的段落
- 重要信息可以使用序号或要点
- 语气温和，不要过于生硬`

	// 构建上下文
	var contextBlock string
	if context != "" {
		contextBlock = fmt.Sprintf("知识库参考:\n%s", context)
	} else {
		contextBlock = "知识库中未找到相关信息，请告知用户暂时无法回答，建议转人工。"
	}

	// 构建用户消息
	userPrompt := fmt.Sprintf(`用户问题: %s

意图识别结果: %s (置信度: %.2f)

%s

请基于以上信息，生成专业、友好的回复。`,
		userMessage,
		intent.Intent,
		intent.Confidence,
		contextBlock,
	)

	// 调用LLM流式API
	fullReply, err := a.LLMClient.ChatWithSystemStream(ctx, systemPrompt, userPrompt, chunkChan)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(fullReply), nil
}

// generateReply 生成回复（非流式）
func (a *Agent) generateReply(userMessage, context string, intent *models.IntentResult, history []models.Message) (string, error) {
	systemPrompt := `你是一个专业、友好的电商客服AI助手。

核心原则：
1. 基于提供的知识库内容回答问题，不要编造信息
2. 如果知识库中没有相关信息，如实告知并建议转人工
3. 回复要简洁、清晰、有温度
4. 使用礼貌用语，称呼用户为"您"

回复格式要求：
- 使用简洁的段落
- 重要信息可以使用序号或要点
- 语气温和，不要过于生硬`

	var contextBlock string
	if context != "" {
		contextBlock = fmt.Sprintf("知识库参考:\n%s", context)
	} else {
		contextBlock = "知识库中未找到相关信息，请告知用户暂时无法回答，建议转人工。"
	}

	userPrompt := fmt.Sprintf(`用户问题: %s

意图识别结果: %s (置信度: %.2f)

%s

请基于以上信息，生成专业、友好的回复。`,
		userMessage,
		intent.Intent,
		intent.Confidence,
		contextBlock,
	)

	reply, err := a.LLMClient.ChatWithSystem(systemPrompt, userPrompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(reply), nil
}
