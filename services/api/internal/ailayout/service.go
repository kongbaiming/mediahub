package ailayout

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mediahub/api/internal/config"
	"github.com/mediahub/api/pkg/logger"
)

// Service AI 布局生成服务
type Service struct {
	cfg    config.AIConfig
	client *http.Client
}

// NewService 构造
func NewService(cfg config.AIConfig) *Service {
	return &Service{
		cfg: cfg,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// GenerateFromText 从文字描述生成布局
func (s *Service) GenerateFromText(ctx context.Context, userPrompt string, stats *LibraryStats) (*LayoutConfigJSON, string, error) {
	libCtx := BuildLibraryContext(stats)
	systemPrompt := fmt.Sprintf(SystemPrompt, libCtx)

	messages := []chatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	raw, err := s.callLLM(ctx, messages)
	if err != nil {
		return nil, "", err
	}

	config, err := parseLayoutJSON(raw)
	if err != nil {
		return nil, "", fmt.Errorf("解析 AI 返回的布局配置失败: %w", err)
	}

	explanation := fmt.Sprintf("根据「%s」生成了 %d 行布局", userPrompt, len(config.Rows))
	return config, explanation, nil
}

// GenerateFromImage 从图片生成布局
func (s *Service) GenerateFromImage(ctx context.Context, imageData []byte, stats *LibraryStats) (*LayoutConfigJSON, string, error) {
	libCtx := BuildLibraryContext(stats)
	systemPrompt := fmt.Sprintf(VisionPrompt, libCtx)

	// base64 编码图片
	b64 := base64.StdEncoding.EncodeToString(imageData)

	messages := []chatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: []contentPart{
			{Type: "image_url", ImageURL: &imageURL{URL: "data:image/jpeg;base64," + b64}},
			{Type: "text", Text: "请分析这张 UI 原型图，生成对应的布局配置。"},
		}},
	}

	raw, err := s.callLLM(ctx, messages)
	if err != nil {
		return nil, "", err
	}

	config, err := parseLayoutJSON(raw)
	if err != nil {
		return nil, "", fmt.Errorf("解析 AI 返回的布局配置失败: %w", err)
	}

	explanation := fmt.Sprintf("从图片识别生成了 %d 行布局", len(config.Rows))
	return config, explanation, nil
}

// ---- LLM 调用 ----

type chatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string 或 []contentPart
}

type contentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (s *Service) callLLM(ctx context.Context, messages []chatMessage) (string, error) {
	switch s.cfg.Provider {
	case "ollama":
		return s.callOllama(ctx, messages)
	default:
		return s.callOpenAI(ctx, messages)
	}
}

func (s *Service) callOpenAI(ctx context.Context, messages []chatMessage) (string, error) {
	baseURL := s.cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	reqBody := chatRequest{
		Model:    s.cfg.Model,
		Messages: messages,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("LLM 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB 限制
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("解析 LLM 响应失败: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("LLM 错误: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("LLM 返回空结果")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (s *Service) callOllama(ctx context.Context, messages []chatMessage) (string, error) {
	baseURL := s.cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	reqBody := map[string]any{
		"model":    s.cfg.Model,
		"messages": messages,
		"stream":   false,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10MB 限制
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("解析 Ollama 响应失败: %w", err)
	}

	return ollamaResp.Message.Content, nil
}

// ---- 解析 ----

func parseLayoutJSON(raw string) (*LayoutConfigJSON, error) {
	// 清理 markdown 代码块标记
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```json") {
		raw = strings.TrimPrefix(raw, "```json")
	} else if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```")
	}
	if strings.HasSuffix(raw, "```") {
		raw = strings.TrimSuffix(raw, "```")
	}
	raw = strings.TrimSpace(raw)

	var config LayoutConfigJSON
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		// 尝试提取 JSON 对象
		start := strings.Index(raw, "{")
		end := strings.LastIndex(raw, "}")
		if start >= 0 && end > start {
			raw = raw[start : end+1]
			if err2 := json.Unmarshal([]byte(raw), &config); err2 != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	// 校验基本结构
	if len(config.Rows) == 0 {
		return nil, fmt.Errorf("AI 生成的布局没有包含任何行")
	}

	// 确保每行有 ID
	for i := range config.Rows {
		if config.Rows[i].ID == "" {
			config.Rows[i].ID = fmt.Sprintf("row-%d", i+1)
		}
	}

	// 默认主题
	if config.Theme == "" {
		config.Theme = "dark"
	}

	logger.Info("AI 布局解析成功", "rows", len(config.Rows), "theme", config.Theme)
	return &config, nil
}
