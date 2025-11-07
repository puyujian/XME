package main

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

// setupRoutes 设置路由配置
func setupRoutes(appServer *AppServer) *gin.Engine {
    // 设置 Gin 模式
    gin.SetMode(gin.ReleaseMode)

    router := gin.New()
    router.Use(gin.Logger())
    router.Use(gin.Recovery())

    // 添加中间件
    router.Use(errorHandlingMiddleware())
    router.Use(corsMiddleware())

    // 静态文件服务 - 提供嵌入的 HTML 文件
    router.GET("/", func(c *gin.Context) {
        content, err := webContent.ReadFile("XhsMcpWeb.html")
        if err != nil {
            c.String(http.StatusInternalServerError, "无法加载网页文件")
            return
        }
        c.Header("Content-Type", "text/html; charset=utf-8")
        c.String(http.StatusOK, string(content))
    })

    // 登录页面
    router.GET("/login.html", func(c *gin.Context) {
        content, err := webContent.ReadFile("login.html")
        if err != nil {
            c.String(http.StatusInternalServerError, "无法加载登录页面")
            return
        }
        c.Header("Content-Type", "text/html; charset=utf-8")
        c.String(http.StatusOK, string(content))
    })

    // 健康检查
    router.GET("/health", healthHandler)

    // MCP 端点 - 使用 Streamable HTTP 协议
    mcpHandler := appServer.StreamableHTTPHandler()
    router.Any("/mcp", gin.WrapH(mcpHandler))
    router.Any("/mcp/*path", gin.WrapH(mcpHandler))

    // API 路由组
    api := router.Group("/api/v1")
    {
        api.GET("/login/status", appServer.checkLoginStatusHandler)
        api.POST("/login", appServer.loginHandler)
        api.GET("/sessions", appServer.listSessionsHandler)
        api.POST("/publish", appServer.publishHandler)
        api.GET("/feeds/list", appServer.listFeedsHandler)
        api.GET("/feeds/search", appServer.searchFeedsHandler)
        
        // AI生成路由
        api.POST("/ai/generate", appServer.aiGenerateHandler)
        
        // 浏览器管理路由
        api.GET("/browser/status", appServer.browserStatusHandler)
        api.POST("/browser/close", appServer.closeBrowserHandler)
        api.POST("/browser/close-all", appServer.closeAllBrowsersHandler)
    }

    return router
}
