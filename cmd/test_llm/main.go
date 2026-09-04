package main

import (
	"fmt"
	"log"

	"github.com/LittleCurry/go_first_ai/internal/config"
	"github.com/LittleCurry/go_first_ai/internal/llm"
)

func main() {
	// 加载配置
	if err := config.LoadConfig(); err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 创建LLM客户端
	client := llm.NewClient(config.AppConfig)

	// 测试1: 简单对话
	fmt.Println("========== 测试1: 简单对话 ==========")
	reply, err := client.SimpleChat("用一句话介绍一下你自己")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("回复: %s\n\n", reply)

	// 测试2: 带系统提示
	fmt.Println("========== 测试2: 带系统提示 ==========")
	systemPrompt := "你是一个专业的电商客服助手，回答要简洁、专业、友好。"
	userMessage := "我下单后多久能发货？"
	reply, err = client.ChatWithSystem(systemPrompt, userMessage)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("回复: %s\n\n", reply)

	fmt.Println("✅ LLM客户端测试完成！")
}
