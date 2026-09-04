package rag

import (
	"fmt"

	"github.com/LittleCurry/go_first_ai/internal/config"
	"github.com/LittleCurry/go_first_ai/internal/models"
	"github.com/LittleCurry/go_first_ai/internal/vector"
)

// Retriever RAG检索器
type Retriever struct {
	ChromaClient *vector.ChromaClient
	Embedder     *Embedder
	TopK         int
	Threshold    float64
}

// NewRetriever 创建检索器
func NewRetriever(cfg *config.Config) *Retriever {
	return &Retriever{
		ChromaClient: vector.NewChromaClient(cfg),
		Embedder:     NewEmbedder("http://localhost:8002"), // 确保是8002端口
		TopK:         cfg.RAG.TopK,
		Threshold:    cfg.RAG.SimilarityThreshold,
	}
}

// SearchResult 检索结果
type SearchResult struct {
	Content  string
	Score    float64
	Metadata map[string]string
}

// Retrieve 检索相关知识
func (r *Retriever) Retrieve(query string) ([]SearchResult, error) {
	// 1. 生成查询向量
	embeddings, err := r.Embedder.GetEmbeddings([]string{query})
	if err != nil {
		return nil, fmt.Errorf("生成向量失败: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("未生成向量")
	}

	// 2. 查询ChromaDB
	resp, err := r.ChromaClient.Query(embeddings, r.TopK)
	if err != nil {
		return nil, fmt.Errorf("查询向量库失败: %w", err)
	}

	// 3. 解析结果
	if len(resp.Documents) == 0 || len(resp.Documents[0]) == 0 {
		return []SearchResult{}, nil
	}

	results := make([]SearchResult, 0, len(resp.Documents[0]))
	for i := range resp.Documents[0] {
		// 计算相似度（距离转相似度）
		similarity := 1.0 - resp.Distances[0][i]

		// 过滤低相似度结果
		if similarity < r.Threshold {
			continue
		}

		meta := make(map[string]string)
		if len(resp.Metadatas) > 0 && len(resp.Metadatas[0]) > i {
			meta = resp.Metadatas[0][i]
		}

		results = append(results, SearchResult{
			Content:  resp.Documents[0][i],
			Score:    similarity,
			Metadata: meta,
		})
	}

	return results, nil
}

// RetrieveForChat 为聊天检索上下文
func (r *Retriever) RetrieveForChat(query string) (string, []models.RAGResult, error) {
	results, err := r.Retrieve(query)
	if err != nil {
		return "", nil, err
	}

	if len(results) == 0 {
		return "", nil, nil
	}

	// 构建上下文
	var context string
	ragResults := make([]models.RAGResult, 0, len(results))

	for i, res := range results {
		// 只取前3条作为上下文
		if i >= 3 {
			break
		}

		context += fmt.Sprintf("【相关知识 %d】\n问题: %s\n答案: %s\n\n",
			i+1,
			res.Metadata["question"],
			res.Content,
		)

		ragResults = append(ragResults, models.RAGResult{
			Content:  res.Content,
			Score:    res.Score,
			Source:   res.Metadata["id"],
			Category: res.Metadata["category"],
		})
	}

	return context, ragResults, nil
}
