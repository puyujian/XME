package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// OpenAIConfig OpenAI配置
type OpenAIConfig struct {
	APIKey      string
	BaseURL     string
	TextModel   string
	VisionModel string
	Timeout     time.Duration
}

// Client AI客户端
type Client struct {
	config     *OpenAIConfig
	httpClient *http.Client
}

// NewClient 创建AI客户端
func NewClient(config *OpenAIConfig) *Client {
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	if config.TextModel == "" {
		config.TextModel = "gpt-4o-mini"
	}

	if config.VisionModel == "" {
		config.VisionModel = "dall-e-3"
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// NewDefaultClient 创建默认配置的AI客户端
func NewDefaultClient() *Client {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		logrus.Warn("OPENAI_API_KEY 环境变量未设置")
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	textModel := os.Getenv("OPENAI_TEXT_MODEL")
	visionModel := os.Getenv("OPENAI_VISION_MODEL")

	return NewClient(&OpenAIConfig{
		APIKey:      apiKey,
		BaseURL:     baseURL,
		TextModel:   textModel,
		VisionModel: visionModel,
	})
}

// ChatCompletionRequest OpenAI聊天请求
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse OpenAI聊天响应
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// ImageGenerationRequest 图片生成请求
type ImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

// ImageGenerationResponse 图片生成响应
type ImageGenerationResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		URL           string `json:"url,omitempty"`
		B64JSON       string `json:"b64_json,omitempty"`
		RevisedPrompt string `json:"revised_prompt,omitempty"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// GenerateText 生成文本
func (c *Client) GenerateText(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	messages := []ChatMessage{}
	
	if systemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}
	
	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: prompt,
	})

	req := ChatCompletionRequest{
		Model:       c.config.TextModel,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   2000,
	}

	return c.chatCompletion(ctx, req)
}

// GenerateImage 生成图片
func (c *Client) GenerateImage(ctx context.Context, prompt string) (string, error) {
	req := ImageGenerationRequest{
		Model:          c.config.VisionModel,
		Prompt:         prompt,
		N:              1,
		Size:           "1024x1024",
		ResponseFormat: "url",
	}

	return c.imageGeneration(ctx, req)
}

// chatCompletion 调用OpenAI聊天完成API
func (c *Client) chatCompletion(ctx context.Context, req ChatCompletionRequest) (string, error) {
	if c.config.APIKey == "" {
		return "", fmt.Errorf("OpenAI API Key未配置")
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", c.config.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))

	logrus.Debugf("调用OpenAI API: %s, model: %s", url, req.Model)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("API返回错误: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API未返回任何结果")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// imageGeneration 调用OpenAI图片生成API
func (c *Client) imageGeneration(ctx context.Context, req ImageGenerationRequest) (string, error) {
	if c.config.APIKey == "" {
		return "", fmt.Errorf("OpenAI API Key未配置")
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("%s/images/generations", c.config.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))

	logrus.Debugf("调用OpenAI图片生成API: %s, model: %s", url, req.Model)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	var imgResp ImageGenerationResponse
	if err := json.Unmarshal(body, &imgResp); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	if imgResp.Error != nil {
		return "", fmt.Errorf("API返回错误: %s", imgResp.Error.Message)
	}

	if len(imgResp.Data) == 0 {
		return "", fmt.Errorf("API未返回任何图片")
	}

	return imgResp.Data[0].URL, nil
}
