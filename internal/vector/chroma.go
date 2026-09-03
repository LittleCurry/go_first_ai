package vector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ChromaClient ChromaDB HTTP客户端
type ChromaClient struct {
	BaseURL    string
	Collection string
	HTTPClient *http.Client
}

// ChromaConfig 配置
type ChromaConfig struct {
	PersistDir string
}

// NewChromaClient 创建客户端
func NewChromaClient(persistDir string) *ChromaClient {
	// Chroma在本地使用HTTP服务，默认端口8001
	// 后面会通过docker-compose启动
	return &ChromaClient{
		BaseURL:    "http://localhost:8001",
		Collection: "customer_service_knowledge",
		HTTPClient: &http.Client{},
	}
}

// QueryRequest 查询请求
type QueryRequest struct {
	QueryEmbeddings [][]float64 `json:"query_embeddings"`
	TopK            int         `json:"n_results"`
}

// QueryResponse 查询响应
type QueryResponse struct {
	IDs       [][]string            `json:"ids"`
	Documents [][]string            `json:"documents"`
	Metadatas [][]map[string]string `json:"metadatas"`
	Distances [][]float64           `json:"distances"`
}

// Query 查询相似向量
func (c *ChromaClient) Query(embeddings [][]float64, topK int) (*QueryResponse, error) {
	url := fmt.Sprintf("%s/api/v1/collections/%s/query", c.BaseURL, c.Collection)

	reqBody := QueryRequest{
		QueryEmbeddings: embeddings,
		TopK:            topK,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chroma error: %s", string(body))
	}

	var result QueryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetCollectionInfo 获取collection信息
func (c *ChromaClient) GetCollectionInfo() (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/collections/%s", c.BaseURL, c.Collection)
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}
