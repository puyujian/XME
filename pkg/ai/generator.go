package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// GenerateRequest AI生成请求
type GenerateRequest struct {
	Topic       string   `json:"topic"`        // 主题
	Keywords    []string `json:"keywords"`     // 关键词
	Style       string   `json:"style"`        // 风格
	ImageCount  int      `json:"image_count"`  // 图片数量
	ContentType string   `json:"content_type"` // 内容类型：article, tutorial, review等
}

// GenerateResult AI生成结果
type GenerateResult struct {
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags"`
	ImageURLs []string `json:"image_urls"`
}

// ContentGeneratorService 内容生成服务
type ContentGeneratorService struct {
	client *Client
}

// NewContentGeneratorService 创建内容生成服务
func NewContentGeneratorService(client *Client) *ContentGeneratorService {
	return &ContentGeneratorService{
		client: client,
	}
}

// GenerateContent 生成完整内容（标题、正文、标签、封面）
func (s *ContentGeneratorService) GenerateContent(ctx context.Context, req *GenerateRequest) (*GenerateResult, error) {
	logrus.Infof("开始生成内容，主题: %s, 关键词: %v", req.Topic, req.Keywords)

	if req.ImageCount == 0 {
		req.ImageCount = 1
	}

	if req.ContentType == "" {
		req.ContentType = "article"
	}

	// 1. 生成标题、内容和标签
	textContent, err := s.generateTextContent(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("生成文本内容失败: %w", err)
	}

	logrus.Infof("生成的标题: %s", textContent.Title)
	logrus.Infof("生成的标签: %v", textContent.Tags)

	// 2. 生成封面图片
	imageURLs := []string{}
	if req.ImageCount > 0 {
		for i := 0; i < req.ImageCount; i++ {
			imagePrompt := s.buildImagePrompt(req, textContent.Title)
			logrus.Infof("生成第 %d 张封面图片，提示词: %s", i+1, imagePrompt)

			imageURL, err := s.client.GenerateImage(ctx, imagePrompt)
			if err != nil {
				logrus.Errorf("生成第 %d 张图片失败: %v", i+1, err)
				// 继续生成其他图片
				continue
			}

			imageURLs = append(imageURLs, imageURL)
			logrus.Infof("第 %d 张图片生成成功: %s", i+1, imageURL)
		}

		if len(imageURLs) == 0 {
			logrus.Warn("所有图片生成失败，但文本内容已生成")
		}
	}

	result := &GenerateResult{
		Title:     textContent.Title,
		Content:   textContent.Content,
		Tags:      textContent.Tags,
		ImageURLs: imageURLs,
	}

	logrus.Infof("内容生成完成，标题: %s, 标签数: %d, 图片数: %d",
		result.Title, len(result.Tags), len(result.ImageURLs))

	return result, nil
}

// TextContent 文本内容
type TextContent struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

// generateTextContent 生成文本内容（标题、正文、标签）
func (s *ContentGeneratorService) generateTextContent(ctx context.Context, req *GenerateRequest) (*TextContent, error) {
	systemPrompt := `你是一个专业的小红书内容创作助手。你需要根据用户提供的主题和关键词，生成吸引人的小红书笔记内容。

要求：
1. 标题要吸引眼球，可以使用emoji，长度控制在30字以内
2. 内容要丰富有价值，适合小红书的风格，可以使用emoji和换行，长度300-800字
3. 标签要相关且热门，数量3-8个，每个标签前加#号
4. 输出格式必须是JSON，包含title、content、tags三个字段

示例输出格式：
{
  "title": "✨标题内容✨",
  "content": "正文内容...",
  "tags": ["#标签1", "#标签2", "#标签3"]
}`

	keywordsStr := strings.Join(req.Keywords, "、")
	if keywordsStr != "" {
		keywordsStr = "，关键词包括：" + keywordsStr
	}

	styleStr := ""
	if req.Style != "" {
		styleStr = "，风格要求：" + req.Style
	}

	contentTypeDesc := ""
	switch req.ContentType {
	case "tutorial":
		contentTypeDesc = "，内容类型为教程类，需要有步骤和指导"
	case "review":
		contentTypeDesc = "，内容类型为评测类，需要客观评价"
	case "experience":
		contentTypeDesc = "，内容类型为体验分享类，需要真实感受"
	}

	userPrompt := fmt.Sprintf("请为主题「%s」生成一篇小红书笔记内容%s%s%s。请直接返回JSON格式，不要添加任何其他说明文字。",
		req.Topic, keywordsStr, styleStr, contentTypeDesc)

	logrus.Debugf("AI文本生成提示词: %s", userPrompt)

	response, err := s.client.GenerateText(ctx, userPrompt, systemPrompt)
	if err != nil {
		return nil, err
	}

	// 清理响应内容，移除可能的markdown代码块标记
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	logrus.Debugf("AI文本生成响应: %s", response)

	// 解析JSON响应
	var textContent TextContent
	if err := json.Unmarshal([]byte(response), &textContent); err != nil {
		return nil, fmt.Errorf("解析AI生成的内容失败: %w, 响应内容: %s", err, response)
	}

	return &textContent, nil
}

// buildImagePrompt 构建图片生成提示词
func (s *ContentGeneratorService) buildImagePrompt(req *GenerateRequest, title string) string {
	// 基础提示词
	prompt := fmt.Sprintf("Create a beautiful and eye-catching cover image for a social media post about: %s", req.Topic)

	// 添加关键词
	if len(req.Keywords) > 0 {
		keywordsEn := strings.Join(req.Keywords, ", ")
		prompt += fmt.Sprintf(". Keywords: %s", keywordsEn)
	}

	// 添加风格要求
	stylePrompt := ", modern and clean style, vibrant colors, professional photography quality"
	if req.Style != "" {
		switch req.Style {
		case "简约":
			stylePrompt = ", minimalist style, clean and simple, pastel colors"
		case "时尚":
			stylePrompt = ", fashionable and trendy style, bold colors, magazine quality"
		case "温馨":
			stylePrompt = ", warm and cozy style, soft colors, inviting atmosphere"
		case "专业":
			stylePrompt = ", professional and business style, corporate colors, high quality"
		default:
			stylePrompt = fmt.Sprintf(", %s style", req.Style)
		}
	}
	prompt += stylePrompt

	// 添加内容类型相关的视觉元素
	switch req.ContentType {
	case "tutorial":
		prompt += ", include educational elements, step-by-step visual hints"
	case "review":
		prompt += ", include product showcase, comparison elements"
	case "experience":
		prompt += ", include lifestyle elements, authentic feel"
	}

	// 添加通用的小红书风格要求
	prompt += ", suitable for Xiaohongshu (Little Red Book) platform, 1:1 aspect ratio"

	return prompt
}

// GenerateTitle 仅生成标题
func (s *ContentGeneratorService) GenerateTitle(ctx context.Context, topic string, keywords []string) (string, error) {
	keywordsStr := ""
	if len(keywords) > 0 {
		keywordsStr = "，关键词：" + strings.Join(keywords, "、")
	}

	systemPrompt := "你是一个小红书标题创作专家，擅长创作吸引人的标题。标题要简洁有力，可以使用emoji，长度控制在30字以内。"
	userPrompt := fmt.Sprintf("请为主题「%s」%s生成一个吸引人的小红书标题。只返回标题文本，不要其他内容。", topic, keywordsStr)

	return s.client.GenerateText(ctx, userPrompt, systemPrompt)
}

// GenerateTags 仅生成标签
func (s *ContentGeneratorService) GenerateTags(ctx context.Context, title string, content string) ([]string, error) {
	systemPrompt := "你是一个小红书标签专家，擅长为内容选择合适的标签。标签要相关且热门，每个标签前加#号。请以JSON数组格式返回标签。"
	userPrompt := fmt.Sprintf("请为以下小红书内容生成3-8个相关标签：\n标题：%s\n内容：%s\n\n请直接返回JSON数组格式，例如：[\"#标签1\", \"#标签2\"]", title, content)

	response, err := s.client.GenerateText(ctx, userPrompt, systemPrompt)
	if err != nil {
		return nil, err
	}

	// 清理响应内容
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var tags []string
	if err := json.Unmarshal([]byte(response), &tags); err != nil {
		return nil, fmt.Errorf("解析标签失败: %w, 响应内容: %s", err, response)
	}

	return tags, nil
}
