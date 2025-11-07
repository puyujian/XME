package main

import (
    "context"
    "os"
    "strings"

    "github.com/sirupsen/logrus"
    "github.com/xpzouying/xiaohongshu-mcp/browser"
    "github.com/xpzouying/xiaohongshu-mcp/configs"
    "github.com/xpzouying/xiaohongshu-mcp/pkg/ai"
    "github.com/xpzouying/xiaohongshu-mcp/pkg/downloader"
    "github.com/xpzouying/xiaohongshu-mcp/xiaohongshu"
)

// XiaohongshuService 小红书业务服务
type XiaohongshuService struct{}

// NewXiaohongshuService 创建小红书服务实例
func NewXiaohongshuService() *XiaohongshuService {
    return &XiaohongshuService{}
}

// PublishRequest 发布请求
type PublishRequest struct {
    Title    string   `json:"title" binding:"required"`
    Content  string   `json:"content" binding:"required"`
    Images   []string `json:"images"`
    Tags     []string `json:"tags,omitempty"`
    Products []string `json:"products,omitempty"`
}

// LoginStatusResponse 登录状态响应
type LoginStatusResponse struct {
    IsLoggedIn bool   `json:"is_logged_in"`
    Username   string `json:"username,omitempty"`
}

// PublishResponse 发布响应
type PublishResponse struct {
    Title   string `json:"title"`
    Content string `json:"content"`
    Images  int    `json:"images"`
    Status  string `json:"status"`
    PostID  string `json:"post_id,omitempty"`
}

// FeedsListResponse Feeds列表响应
type FeedsListResponse struct {
    Feeds []xiaohongshu.Feed `json:"feeds"`
    Count int                `json:"count"`
}

// CheckLoginStatus 检查登录状态
func (s *XiaohongshuService) CheckLoginStatus(ctx context.Context) (*LoginStatusResponse, error) {
    b := browser.NewBrowser(configs.IsHeadless())
    // 注意：不再自动关闭浏览器，由浏览器管理器管理生命周期

    page := b.NewPage()
    defer page.Close()

    loginAction := xiaohongshu.NewLogin(page)

    isLoggedIn, err := loginAction.CheckLoginStatus(ctx)
    if err != nil {
        return nil, err
    }

    response := &LoginStatusResponse{
        IsLoggedIn: isLoggedIn,
        Username:   configs.Username,
    }

    return response, nil
}

// PublishContent 发布内容
func (s *XiaohongshuService) PublishContent(ctx context.Context, req *PublishRequest) (*PublishResponse, error) {
    logrus.Infof("开始处理发布请求: 标题=%s, 图片数量=%d, 标签数量=%d, 商品数量=%d", req.Title, len(req.Images), len(req.Tags), len(req.Products))

    var imagePaths []string

    if len(req.Images) == 0 {
        logrus.Warn("没有提供图片，将尝试发布纯文本内容")
    } else {
        // 检查是否是虚拟图片路径（前端生成的临时路径）
        virtualImagePaths := make([]string, 0, len(req.Images))
        for _, img := range req.Images {
            isVirtual := false

            // 模式1: image_xxx_xxx.jpg
            if strings.HasPrefix(img, "image_") && strings.Contains(img, "_") {
                isVirtual = true
            }

            // 模式2: 检查是否是简单的文件名（不包含路径分隔符）
            if !isVirtual && !strings.Contains(img, "/") && !strings.Contains(img, "\\") {
                if _, err := os.Stat(img); os.IsNotExist(err) {
                    isVirtual = true
                }
            }

            // 模式3: 检查是否是URL（以http开头）
            if !isVirtual && (strings.HasPrefix(img, "http://") || strings.HasPrefix(img, "https://")) {
                isVirtual = false // URL不是虚拟路径
            }

            if isVirtual {
                virtualImagePaths = append(virtualImagePaths, img)
            }
        }

        if len(virtualImagePaths) == len(req.Images) {
            logrus.Warnf("检测到虚拟图片路径: %v，将发布纯文本内容", virtualImagePaths)
        } else {
            processed, err := s.processImages(req.Images)
            if err != nil {
                logrus.Errorf("图片处理失败: %v", err)
                return nil, err
            }
            imagePaths = processed
            logrus.Infof("图片处理完成，有效图片数量: %d", len(imagePaths))
        }
    }

    // 构建发布内容
    content := xiaohongshu.PublishImageContent{
        Title:      req.Title,
        Content:    req.Content,
        Tags:       req.Tags,
        Products:   req.Products,
        ImagePaths: imagePaths,
    }

    // 执行发布
    if err := s.publishContent(ctx, content); err != nil {
        logrus.Errorf("发布内容执行失败: %v", err)
        return nil, err
    }

    status := "发布完成"
    if len(imagePaths) == 0 {
        status = "发布完成（纯文本）"
    }

    response := &PublishResponse{
        Title:   req.Title,
        Content: req.Content,
        Images:  len(imagePaths),
        Status:  status,
    }

    logrus.Infof("发布内容处理完成: %+v", response)
    return response, nil
}

// processImages 处理图片列表，支持URL下载和本地路径
func (s *XiaohongshuService) processImages(images []string) ([]string, error) {
    processor := downloader.NewImageProcessor()
    return processor.ProcessImages(images)
}

// publishContent 执行内容发布
func (s *XiaohongshuService) publishContent(ctx context.Context, content xiaohongshu.PublishImageContent) error {
    logrus.Infof("开始执行发布，使用环境变量 MCP_HEADLESS: %s", os.Getenv("MCP_HEADLESS"))

    // 使用浏览器管理器的当前设置
    manager := browser.GetManager()
    currentHeadless := manager.IsHeadless()
    b := browser.NewBrowser(currentHeadless) // 使用当前的无头模式设置
    // 注意：不再自动关闭浏览器，由浏览器管理器管理生命周期

    page := b.NewPage()
    defer page.Close()

    action, err := xiaohongshu.NewPublishImageAction(page)
    if err != nil {
        logrus.Errorf("创建发布action失败: %v", err)
        return err
    }

    // 执行发布
    logrus.Info("开始执行发布操作...")
    if err := action.Publish(ctx, content); err != nil {
        logrus.Errorf("发布操作失败: %v", err)
        return err
    }

    logrus.Info("发布操作完成")
    return nil
}

// ListFeeds 获取Feeds列表
func (s *XiaohongshuService) ListFeeds(ctx context.Context) (*FeedsListResponse, error) {
    // 使用浏览器管理器的当前设置
    manager := browser.GetManager()
    currentHeadless := manager.IsHeadless()
    b := browser.NewBrowser(currentHeadless) // 使用当前的无头模式设置
    // 注意：不再自动关闭浏览器，由浏览器管理器管理生命周期

    page := b.NewPage()
    defer page.Close()

    // 创建 Feeds 列表 action
    action := xiaohongshu.NewFeedsListAction(page)

    // 获取 Feeds 列表
    feeds, err := action.GetFeedsList(ctx)
    if err != nil {
        return nil, err
    }

    response := &FeedsListResponse{
        Feeds: feeds,
        Count: len(feeds),
    }

    return response, nil
}

func (s *XiaohongshuService) SearchFeeds(ctx context.Context, keyword string) (*FeedsListResponse, error) {
    // 使用浏览器管理器的当前设置
    manager := browser.GetManager()
    currentHeadless := manager.IsHeadless()
    b := browser.NewBrowser(currentHeadless) // 使用当前的无头模式设置
    // 注意：不再自动关闭浏览器，由浏览器管理器管理生命周期

    page := b.NewPage()
    defer page.Close()

    action := xiaohongshu.NewSearchAction(page)

    feeds, err := action.Search(ctx, keyword)
    if err != nil {
        return nil, err
    }

    response := &FeedsListResponse{
        Feeds: feeds,
        Count: len(feeds),
    }

    return response, nil
}

// AIGenerateRequest AI生成请求
type AIGenerateRequest struct {
    Topic       string   `json:"topic" binding:"required"`
    Keywords    []string `json:"keywords,omitempty"`
    Style       string   `json:"style,omitempty"`
    ImageCount  int      `json:"image_count,omitempty"`
    ContentType string   `json:"content_type,omitempty"`
    AutoPublish bool     `json:"auto_publish,omitempty"`
}

// AIGenerateResponse AI生成响应
type AIGenerateResponse struct {
    Title     string   `json:"title"`
    Content   string   `json:"content"`
    Tags      []string `json:"tags"`
    ImageURLs []string `json:"image_urls"`
    Status    string   `json:"status"`
    PostID    string   `json:"post_id,omitempty"`
}

// AIGenerateContent AI生成内容
func (s *XiaohongshuService) AIGenerateContent(ctx context.Context, req *AIGenerateRequest) (*AIGenerateResponse, error) {
    logrus.Infof("开始AI生成内容: 主题=%s, 关键词=%v, 风格=%s, 图片数量=%d, 内容类型=%s",
        req.Topic, req.Keywords, req.Style, req.ImageCount, req.ContentType)

    // 创建AI客户端
    aiClient := ai.NewDefaultClient()
    generator := ai.NewContentGeneratorService(aiClient)

    // 构建生成请求
    genReq := &ai.GenerateRequest{
        Topic:       req.Topic,
        Keywords:    req.Keywords,
        Style:       req.Style,
        ImageCount:  req.ImageCount,
        ContentType: req.ContentType,
    }

    // 生成内容
    result, err := generator.GenerateContent(ctx, genReq)
    if err != nil {
        logrus.Errorf("AI生成内容失败: %v", err)
        return nil, err
    }

    logrus.Infof("AI生成完成: 标题=%s, 标签=%v, 图片数量=%d", result.Title, result.Tags, len(result.ImageURLs))

    response := &AIGenerateResponse{
        Title:     result.Title,
        Content:   result.Content,
        Tags:      result.Tags,
        ImageURLs: result.ImageURLs,
        Status:    "生成完成",
    }

    return response, nil
}
