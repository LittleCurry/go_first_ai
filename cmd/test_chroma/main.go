package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/LittleCurry/go_first_ai/internal/config"
	"github.com/LittleCurry/go_first_ai/internal/vector"
)

func main() {
	// 加载配置
	if err := config.LoadConfig(); err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 创建Chroma客户端
	client := vector.NewChromaClient(config.AppConfig)

	// 测试1: 健康检查
	fmt.Println("========== 测试1: 健康检查 ==========")
	if err := client.HealthCheck(); err != nil {
		log.Fatal("❌ Chroma服务不可用:", err)
	}
	fmt.Println("✅ Chroma服务正常")

	// 测试2: 列出所有collection
	fmt.Println("\n========== 测试2: 列出所有collection ==========")
	collections, err := client.ListCollections()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("共 %d 个collection\n", len(collections))
	for _, col := range collections {
		fmt.Printf("  - %s\n", col["name"])
	}

	// 测试3: 获取指定collection信息
	fmt.Println("\n========== 测试3: 获取collection信息 ==========")
	info, err := client.GetCollectionInfo()
	if err != nil {
		log.Fatal("❌ 获取collection信息失败:", err)
	}
	pretty, _ := json.MarshalIndent(info, "", "  ")
	fmt.Printf("Collection信息:\n%s\n", string(pretty))

	// 测试4: 获取文档数量
	fmt.Println("\n========== 测试4: 获取文档数量 ==========")
	count, err := client.Count()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ 知识库中共有 %d 条文档\n", count)
}
