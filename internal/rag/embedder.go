package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Embedder 生成向量的服务
type Embedder struct {
	APIURL string
	Client *http.Client
}

// NewEmbedder 创建Embedder
func NewEmbedder(apiURL string) *Embedder {
	return &Embedder{
		APIURL: apiURL,
		Client: &http.Client{},
	}
}

// EmbedRequest 请求
type EmbedRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// EmbedResponse 响应
type EmbedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// GetEmbeddings 获取文本向量
func (e *Embedder) GetEmbeddings(texts []string) ([][]float64, error) {
	reqBody := EmbedRequest{
		Input: texts,
		Model: "BAAI/bge-small-zh-v1.5", // 通过本地服务调用
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := e.Client.Post(e.APIURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding error: %s", string(body))
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
