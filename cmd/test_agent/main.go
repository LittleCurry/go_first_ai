package main

import (
	"fmt"
	"log"

	"github.com/LittleCurry/go_first_ai/internal/agent"
	"github.com/LittleCurry/go_first_ai/internal/config"
	"github.com/LittleCurry/go_first_ai/internal/models"
)

func main() {
	// 加载配置
	if err := config.LoadConfig(); err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 创建Agent
	ag := agent.NewAgent(config.AppConfig)

	fmt.Println("========== AI智能客服 Agent 测试 ==========")
	fmt.Println()

	// 测试用例
	testCases := []string{
		"怎么查我的快递？",
		"我想退货，怎么操作？",
		"密码忘了怎么办？",
		"你们有什么优惠活动吗？",
	}

	for _, query := range testCases {
		fmt.Printf("用户: %s\n", query)

		resp, err := ag.Process(query, []models.Message{})
		if err != nil {
			fmt.Printf("❌ 错误: %v\n\n", err)
			continue
		}

		fmt.Printf("AI: %s\n", resp.Reply)
		fmt.Printf("意图: %s (置信度: %.2f)\n", resp.Intent, resp.Confidence)
		if len(resp.Sources) > 0 {
			fmt.Printf("引用知识: %v\n", resp.Sources)
		}
		if resp.NeedTransfer {
			fmt.Printf("⚠️ 需要转人工: %s\n", resp.Reason)
		}
		fmt.Println()
	}
}
