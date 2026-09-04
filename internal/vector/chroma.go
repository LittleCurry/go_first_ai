package vector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/LittleCurry/go_first_ai/internal/config"
)

// ChromaClient ChromaDB HTTP客户端（v2 API）
type ChromaClient struct {
	BaseURL      string
	Tenant       string
	Database     string
	CollectionID string // 直接使用UUID
	HTTPClient   *http.Client
}

// NewChromaClient 创建客户端
func NewChromaClient(cfg *config.Config) *ChromaClient {
	return &ChromaClient{
		BaseURL:      "http://localhost:8001",
		Tenant:       "default_tenant",
		Database:     "default_database",
		CollectionID: "b2b3048b-4151-4b9b-9433-cf0319469d17", // 直接使用UUID
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// getCollectionURL 获取collection的完整CRN URL（使用UUID）
func (c *ChromaClient) getCollectionURL() string {
	return fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections/%s",
		c.BaseURL, c.Tenant, c.Database, c.CollectionID)
}

// getCollectionsURL 获取所有collection的URL
func (c *ChromaClient) getCollectionsURL() string {
	return fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections",
		c.BaseURL, c.Tenant, c.Database)
}

// HealthCheck 检查服务是否可用
func (c *ChromaClient) HealthCheck() error {
	url := c.BaseURL + "/api/v2/heartbeat"
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("chroma service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chroma service error: status %d", resp.StatusCode)
	}
	return nil
}

// Count 获取collection中的文档数量
func (c *ChromaClient) Count() (int, error) {
	url := c.getCollectionURL() + "/count"

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("chroma error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	return result.Count, nil
}

// QueryRequest v2查询请求
type QueryRequest struct {
	Tenant          string                 `json:"tenant,omitempty"`
	Database        string                 `json:"database,omitempty"`
	QueryEmbeddings [][]float64            `json:"query_embeddings"`
	TopK            int                    `json:"n_results"`
	Where           map[string]interface{} `json:"where,omitempty"`
	Include         []string               `json:"include,omitempty"`
}

// QueryResponse v2查询响应
type QueryResponse struct {
	IDs       [][]string            `json:"ids"`
	Documents [][]string            `json:"documents"`
	Metadatas [][]map[string]string `json:"metadatas"`
	Distances [][]float64           `json:"distances"`
}

// Query 查询相似向量
func (c *ChromaClient) Query(embeddings [][]float64, topK int) (*QueryResponse, error) {
	url := c.getCollectionURL() + "/query"

	reqBody := QueryRequest{
		Tenant:          c.Tenant,
		Database:        c.Database,
		QueryEmbeddings: embeddings,
		TopK:            topK,
		Include:         []string{"documents", "metadatas", "distances"},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	fmt.Printf("🔍 ChromaDB Query URL: %s\n", url)

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
		return nil, fmt.Errorf("chroma error (status %d): %s", resp.StatusCode, string(body))
	}

	var result QueryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetCollectionInfo 获取collection信息
func (c *ChromaClient) GetCollectionInfo() (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v2/tenants/%s/databases/%s/collections/%s",
		c.BaseURL, c.Tenant, c.Database, c.CollectionID)
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chroma error (status %d): %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ListCollections 列出所有collection
func (c *ChromaClient) ListCollections() ([]map[string]interface{}, error) {
	url := c.getCollectionsURL()
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chroma error (status %d): %s", resp.StatusCode, string(body))
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}
