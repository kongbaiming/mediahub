package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config Prowlarr 索引器配置
type Config struct {
	URL    string
	APIKey string
}

// Client Prowlarr REST API 客户端
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewClient 构造
func NewClient(cfg Config) *Client {
	return &Client{
		baseURL: strings.TrimRight(cfg.URL, "/"),
		apiKey:  strings.TrimSpace(cfg.APIKey),
		http:    &http.Client{Timeout: 45 * time.Second},
	}
}

// Enabled 是否已配置索引器
func (c *Client) Enabled() bool {
	return c != nil && c.baseURL != "" && c.apiKey != ""
}

// Search 搜索资源
func (c *Client) Search(ctx context.Context, query, searchType string, limit int) ([]Release, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("索引器未配置")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("搜索词不能为空")
	}
	if limit <= 0 {
		limit = 25
	}
	if searchType == "" {
		searchType = "search"
	}

	q := url.Values{}
	q.Set("query", query)
	q.Set("type", searchType)
	q.Set("limit", fmt.Sprintf("%d", limit))

	reqURL := fmt.Sprintf("%s/api/v1/search?%s", c.baseURL, q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("索引器返回 HTTP %d", resp.StatusCode)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, err
	}
	return releases, nil
}
