package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/browser"
	"github.com/xpzouying/xiaohongshu-mcp/cookies"
	"github.com/xpzouying/xiaohongshu-mcp/xiaohongshu"
)

// respondError 返回错误响应
func respondError(c *gin.Context, statusCode int, code, message string, details any) {
	response := ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}

	logrus.Errorf("%s %s %s %d", c.Request.Method, c.Request.URL.Path,
		c.GetString("account"), statusCode)

	c.JSON(statusCode, response)
}

// respondSuccess 返回成功响应
func respondSuccess(c *gin.Context, data any, message string) {
	response := SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	}

	logrus.Infof("%s %s %s %d", c.Request.Method, c.Request.URL.Path,
		c.GetString("account"), http.StatusOK)

	c.JSON(http.StatusOK, response)
}

// setSessionFromRequest 读取请求中的会话并设置到环境变量（供 browser 读取）
func setSessionFromRequest(c *gin.Context) {
	if sid := c.GetHeader("Mcp-Session-Id"); sid != "" {
		_ = os.Setenv("MCP_SESSION_ID", sid)
	}
	if sid := c.Query("session_id"); sid != "" {
		_ = os.Setenv("MCP_SESSION_ID", sid)
	}

	// 处理无头模式设置
	if hv := c.GetHeader("Mcp-Headless"); hv != "" {
		_ = os.Setenv("MCP_HEADLESS", hv)

		// 同时更新浏览器管理器的设置
		if b, err := parseBool(hv); err == nil {
			manager := browser.GetManager()
			manager.SetHeadless(b)
			logrus.Debugf("无头模式已更新: %v", b)
		}
	}
}

// checkLoginStatusHandler 检查登录状态
func (s *AppServer) checkLoginStatusHandler(c *gin.Context) {
	setSessionFromRequest(c)
	status, err := s.xiaohongshuService.CheckLoginStatus(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "STATUS_CHECK_FAILED",
			"检查登录状态失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, status, "检查登录状态成功")
}

// publishHandler 发布内容
func (s *AppServer) publishHandler(c *gin.Context) {
	setSessionFromRequest(c)
	var req PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logrus.Errorf("发布请求参数解析失败: %v", err)
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST",
			"请求参数错误", err.Error())
		return
	}

	logrus.Infof("收到发布请求: 标题=%s, 内容长度=%d, 图片数量=%d, 标签数量=%d, 商品数量=%d",
		req.Title, len(req.Content), len(req.Images), len(req.Tags), len(req.Products))

	// 执行发布
	result, err := s.xiaohongshuService.PublishContent(c.Request.Context(), &req)
	if err != nil {
		logrus.Errorf("发布内容失败: %v", err)
		respondError(c, http.StatusInternalServerError, "PUBLISH_FAILED",
			"发布失败", err.Error())
		return
	}

	logrus.Infof("发布成功: %+v", result)
	respondSuccess(c, result, "发布成功")
}

// listFeedsHandler 获取Feeds列表
func (s *AppServer) listFeedsHandler(c *gin.Context) {
	setSessionFromRequest(c)
	// 获取 Feeds 列表
	result, err := s.xiaohongshuService.ListFeeds(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, "LIST_FEEDS_FAILED",
			"获取Feeds列表失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, result, "获取Feeds列表成功")
}

// searchFeedsHandler 搜索Feeds
func (s *AppServer) searchFeedsHandler(c *gin.Context) {
	setSessionFromRequest(c)
	keyword := c.Query("keyword")
	if keyword == "" {
		respondError(c, http.StatusBadRequest, "MISSING_KEYWORD",
			"缺少关键词参数", "keyword parameter is required")
		return
	}

	// 搜索 Feeds
	result, err := s.xiaohongshuService.SearchFeeds(c.Request.Context(), keyword)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "SEARCH_FEEDS_FAILED",
			"搜索Feeds失败", err.Error())
		return
	}

	c.Set("account", "ai-report")
	respondSuccess(c, result, "搜索Feeds成功")
}

// healthHandler 健康检查
func healthHandler(c *gin.Context) {
	respondSuccess(c, map[string]any{
		"status":    "healthy",
		"service":   "xiaohongshu-mcp",
		"account":   "ai-report",
		"timestamp": "now",
	}, "服务正常")
}

// LoginRequest 登录请求结构
type LoginRequest struct {
	SessionName string `json:"session_name"`
}

// loginHandler 处理登录请求
func (s *AppServer) loginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有JSON数据，使用默认值
		req.SessionName = ""
	}

	// 异步执行登录流程
	go func() {
		logrus.Infof("开始执行自动登录，Session名称: %s", req.SessionName)

		// 直接调用登录逻辑
		if err := s.performLogin(req.SessionName); err != nil {
			logrus.Errorf("登录失败: %v", err)
		} else {
			logrus.Info("登录成功")
		}
	}()

	respondSuccess(c, map[string]interface{}{
		"message":      "登录流程已启动，请等待浏览器自动打开登录页面",
		"status":       "started",
		"session_name": req.SessionName,
	}, "登录流程已启动")
}

// performLogin 执行登录逻辑
func (s *AppServer) performLogin(sessionName string) error {
	// 关闭 go-rod 的 leakless，避免被 Windows Defender 误杀
	os.Setenv("ROD_LAUNCH_LEAKLESS", "0")

	// 创建浏览器实例（非无头模式，用于登录）
	b := browser.NewBrowser(false)
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

	// 创建登录action
	action := xiaohongshu.NewLogin(page)

	// 检查当前登录状态
	status, err := action.CheckLoginStatus(context.Background())
	if err != nil {
		return fmt.Errorf("检查登录状态失败: %v", err)
	}

	logrus.Infof("当前登录状态: %v", status)

	// 无论是否已登录，都保存一次 cookies，确保 ./cookies 目录与文件创建成功
	if err := s.saveCookies(page, sessionName); err != nil {
		logrus.Warnf("保存 cookies 失败（将继续流程）：%v", err)
	}

	if status {
		logrus.Info("已登录，已写入 cookies 文件")
		return nil
	}

	// 开始登录流程
	logrus.Info("开始登录流程...")
	if err = action.Login(context.Background()); err != nil {
		return fmt.Errorf("登录失败: %v", err)
	}

	// 保存cookies
	if err := s.saveCookies(page, sessionName); err != nil {
		return fmt.Errorf("保存cookies失败: %v", err)
	}

	// 再次检查登录状态确认成功
	status, err = action.CheckLoginStatus(context.Background())
	if err != nil {
		return fmt.Errorf("登录后检查状态失败: %v", err)
	}

	if status {
		logrus.Info("登录成功！")
		return nil
	} else {
		return fmt.Errorf("登录流程完成但仍未登录")
	}
}

// saveCookies 保存cookies
func (s *AppServer) saveCookies(page *rod.Page, sessionName string) error {
	cks, err := page.Browser().GetCookies()
	if err != nil {
		return err
	}

	data, err := json.Marshal(cks)
	if err != nil {
		return err
	}

	// 根据session名称决定保存路径
	var path string
	if sessionName != "" {
		// 使用自定义session名称
		path = cookies.GetCookiesFilePathWithSession(sessionName)
	} else {
		// 自动生成session名称
		path = cookies.GetCookiePathForSaving()
	}

	logrus.Infof("保存cookies到: %s", path)
	cookieLoader := cookies.NewLoadCookie(path)
	return cookieLoader.SaveCookies(data)
}

// listSessionsHandler 列出本地会话（cookies）
func (s *AppServer) listSessionsHandler(c *gin.Context) {
	list, err := cookies.ListSessions()
	if err != nil {
		respondError(c, http.StatusInternalServerError, "LIST_SESSIONS_FAILED", "获取会话列表失败", err.Error())
		return
	}
	respondSuccess(c, gin.H{"sessions": list}, "获取会话列表成功")
}

// browserStatusHandler 获取浏览器状态
func (s *AppServer) browserStatusHandler(c *gin.Context) {
	manager := browser.GetManager()

	status := map[string]interface{}{
		"headless_mode":   manager.IsHeadless(),
		"session_count":   manager.GetSessionCount(),
		"active_sessions": []string{}, // 可以扩展显示活跃会话列表
	}

	respondSuccess(c, status, "浏览器状态获取成功")
}

// closeBrowserHandler 关闭指定会话的浏览器
func (s *AppServer) closeBrowserHandler(c *gin.Context) {
	var req struct {
		SessionID string `json:"session_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "无效的请求参数", err.Error())
		return
	}

	if req.SessionID == "" {
		respondError(c, http.StatusBadRequest, "MISSING_SESSION_ID", "缺少会话ID", nil)
		return
	}

	manager := browser.GetManager()
	manager.CloseBrowser(req.SessionID)

	respondSuccess(c, map[string]string{"session_id": req.SessionID}, "浏览器已关闭")
}

// closeAllBrowsersHandler 关闭所有浏览器
func (s *AppServer) closeAllBrowsersHandler(c *gin.Context) {
	manager := browser.GetManager()
	manager.CloseAll()

	respondSuccess(c, map[string]int{"closed_count": manager.GetSessionCount()}, "所有浏览器已关闭")
}

// parseBool 解析布尔值字符串
func parseBool(s string) (bool, error) {
	switch s {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	default:
		return false, nil
	}
}
