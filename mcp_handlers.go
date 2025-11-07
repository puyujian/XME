package main

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/sirupsen/logrus"
)

// MCP 工具处理函数

// handleCheckLoginStatus 处理检查登录状态
func (s *AppServer) handleCheckLoginStatus(ctx context.Context) *MCPToolResult {
    logrus.Info("MCP: 检查登录状态")

    status, err := s.xiaohongshuService.CheckLoginStatus(ctx)
    if err != nil {
        return &MCPToolResult{
            Content: []MCPContent{{
                Type: "text",
                Text: "检查登录状态失败: " + err.Error(),
            }},
            IsError: true,
        }
    }

    resultText := fmt.Sprintf("登录状态检查成功: %+v", status)
    return &MCPToolResult{
        Content: []MCPContent{{
            Type: "text",
            Text: resultText,
        }},
    }
}

// handlePublishContent 处理发布内容
func (s *AppServer) handlePublishContent(ctx context.Context, args map[string]interface{}) *MCPToolResult {
    logrus.Info("MCP: 发布内容")

    // 解析参数
    title, _ := args["title"].(string)
    content, _ := args["content"].(string)
    imagePathsInterface, _ := args["images"].([]interface{})
    tagsInterface, _ := args["tags"].([]interface{})
    productsInterface, _ := args["products"].([]interface{})

    var imagePaths []string
    for _, path := range imagePathsInterface {
        if pathStr, ok := path.(string); ok {
            imagePaths = append(imagePaths, pathStr)
        }
    }

    var tags []string
    for _, tag := range tagsInterface {
        if tagStr, ok := tag.(string); ok {
            tags = append(tags, tagStr)
        }
    }

    var products []string
    for _, product := range productsInterface {
        if productStr, ok := product.(string); ok {
            products = append(products, productStr)
        }
    }

    logrus.Infof("MCP: 发布内容 - 标题: %s, 图片数量: %d, 标签数量: %d, 商品数量: %d",
        title, len(imagePaths), len(tags), len(products))

    // 构建发布请求
    req := &PublishRequest{
        Title:    title,
        Content:  content,
        Images:   imagePaths,
        Tags:     tags,
        Products: products,
    }

    // 执行发布
    result, err := s.xiaohongshuService.PublishContent(ctx, req)
    if err != nil {
        return &MCPToolResult{
            Content: []MCPContent{{
                Type: "text",
                Text: "发布失败: " + err.Error(),
            }},
            IsError: true,
        }
    }

    resultText := fmt.Sprintf("内容发布成功: %+v", result)
    return &MCPToolResult{
        Content: []MCPContent{{
            Type: "text",
            Text: resultText,
        }},
    }
}

// handleListFeeds 处理获取Feeds列表
func (s *AppServer) handleListFeeds(ctx context.Context) *MCPToolResult {
    logrus.Info("MCP: 获取Feeds列表")

    result, err := s.xiaohongshuService.ListFeeds(ctx)
    if err != nil {
        return &MCPToolResult{
            Content: []MCPContent{{
                Type: "text",
                Text: "获取Feeds列表失败: " + err.Error(),
            }},
            IsError: true,
        }
    }

    // 格式化输出，转换为JSON字符串
    jsonData, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        return &MCPToolResult{
            Content: []MCPContent{{
                Type: "text",
                Text: fmt.Sprintf("获取Feeds列表成功，但序列化失败: %v", err),
            }},
            IsError: true,
        }
    }

    return &MCPToolResult{
        Content: []MCPContent{{
            Type: "text",
            Text: string(jsonData),
        }},
    }
}

// handleSearchFeeds 处理搜索Feeds
func (s *AppServer) handleSearchFeeds(ctx context.Context, args map[string]interface{}) *MCPToolResult {
    logrus.Info("MCP: 搜索Feeds")

    // 解析参数
    keyword, ok := args["keyword"].(string)
    if !ok || keyword == "" {
        return &MCPToolResult{
            Content: []MCPContent{{
                Type: "text",
                Text: "搜索Feeds失败: 缺少关键词参数",
            }},
            IsError: true,
        }
    }

    logrus.Infof("MCP: 搜索Feeds - 关键词: %s", keyword)

    result, err := s.xiaohongshuService.SearchFeeds(ctx, keyword)
    if err != nil {
        return &MCPToolResult{
            Content: []MCPContent{{
                Type: "text",
                Text: "搜索Feeds失败: " + err.Error(),
            }},
            IsError: true,
        }
    }

    // 格式化输出，转换为JSON字符串
    jsonData, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        return &MCPToolResult{
            Content: []MCPContent{{
                Type: "text",
                Text: fmt.Sprintf("搜索Feeds成功，但序列化失败: %v", err),
            }},
            IsError: true,
        }
    }

    return &MCPToolResult{
        Content: []MCPContent{{
            Type: "text",
            Text: string(jsonData),
        }},
    }
}

// handleAIGenerate 处理AI生成内容
func (s *AppServer) handleAIGenerate(ctx context.Context, args map[string]interface{}) *MCPToolResult {
    logrus.Info("MCP: AI生成内容")

    // 解析参数
    topic, ok := args["topic"].(string)
    if !ok || topic == "" {
        return &MCPToolResult{
            Content: []MCPContent{{
                Type: "text",
                Text: "AI生成失败: 缺少主题参数",
            }},
            IsError: true,
        }
    }

    // 解析关键词
    var keywords []string
    if keywordsInterface, ok := args["keywords"].([]interface{}); ok {
        for _, kw := range keywordsInterface {
            if kwStr, ok := kw.(string); ok {
                keywords = append(keywords, kwStr)
            }
        }
    }

    // 解析其他参数
    style, _ := args["style"].(string)
    contentType, _ := args["content_type"].(string)
    autoPublish, _ := args["auto_publish"].(bool)

    imageCount := 1
    if ic, ok := args["image_count"].(float64); ok {
        imageCount = int(ic)
    }

    logrus.Infof("MCP: AI生成 - 主题: %s, 关键词: %v, 风格: %s, 图片数量: %d, 内容类型: %s, 自动发布: %v",
        topic, keywords, style, imageCount, contentType, autoPublish)

    // 构建请求
    req := &AIGenerateRequest{
        Topic:       topic,
        Keywords:    keywords,
        Style:       style,
        ImageCount:  imageCount,
        ContentType: contentType,
    }

    // 执行生成
    result, err := s.xiaohongshuService.AIGenerateContent(ctx, req)
    if err != nil {
        return &MCPToolResult{
            Content: []MCPContent{{
                Type: "text",
                Text: "AI生成失败: " + err.Error(),
            }},
            IsError: true,
        }
    }

    // 如果需要自动发布
    if autoPublish {
        logrus.Info("开始自动发布生成的内容...")
        publishReq := &PublishRequest{
            Title:   result.Title,
            Content: result.Content,
            Images:  result.ImageURLs,
            Tags:    result.Tags,
        }

        publishResult, err := s.xiaohongshuService.PublishContent(ctx, publishReq)
        if err != nil {
            logrus.Errorf("自动发布失败: %v", err)
            result.Status = "生成完成，但发布失败: " + err.Error()
        } else {
            result.Status = "生成并发布完成"
            result.PostID = publishResult.PostID
            logrus.Infof("自动发布成功: %+v", publishResult)
        }
    }

    // 格式化输出
    jsonData, err := json.MarshalIndent(result, "", "  ")
    if err != nil {
        return &MCPToolResult{
            Content: []MCPContent{{
                Type: "text",
                Text: fmt.Sprintf("AI生成成功，但序列化失败: %v", err),
            }},
            IsError: true,
        }
    }

    return &MCPToolResult{
        Content: []MCPContent{{
            Type: "text",
            Text: string(jsonData),
        }},
    }
}
