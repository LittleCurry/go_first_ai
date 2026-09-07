package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Embedder 生成向量的服务
type Embedder struct {
	BaseURL string
	Client  *http.Client
}

// NewEmbedder 创建Embedder
func NewEmbedder(baseURL string) *Embedder {
	if baseURL == "" {
		baseURL = "http://localhost:8002"
	}
	return &Embedder{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EmbedRequest 请求
type EmbedRequest struct {
	Input []string `json:"input"`
}

// EmbedResponse 响应
type EmbedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// GetEmbeddings 获取文本向量
func (e *Embedder) GetEmbeddings(texts []string) ([][]float64, error) {
	// 注意：URL是 /embed，不是 /
	url := e.BaseURL + "/embed"

	//fmt.Printf("🔍 请求Embedding URL: %s\n", url) // 添加这行调试

	reqBody := EmbedRequest{
		Input: texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := e.Client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding error (status %d): %s", resp.StatusCode, string(body))
	}

	var result EmbedResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	embeddings := make([][]float64, len(result.Data))
	for i, d := range result.Data {
		embeddings[i] = d.Embedding
	}

	return embeddings, nil
}

// HealthCheck 检查服务是否可用
func (e *Embedder) HealthCheck() error {
	url := e.BaseURL + "/health"
	resp, err := e.Client.Get(url)
	if err != nil {
		return fmt.Errorf("embedding service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("embedding service error: status %d", resp.StatusCode)
	}
	return nil
}
