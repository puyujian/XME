## 🎉 Electron + GO 开发的小红书多账号管理神器


### ✅ 已完成的功能

1. **发帖**
2. **查询内容**
3. **获取主页信息流**
4. **支持多账号**
5. **发布携带商品的图文笔记**（详情见 [商品笔记发布功能](./PRODUCT_FEATURE.md)）


界面展示

![](./png/image.png)


问题反馈
![](./png/af001129f09862c6c491203948b6a29f.jpg)


赞赏

![](./png/6fbe97d18b0a6992141ece2aadea0a9d.jpg)


### 打包教程

1. 打包 go 服务端 为 exe

```bash
go build -ldflags "-s -w" -o dist/backend/xiaohongshu-mcp.exe .
```

2. 打包 exe 主程序 

```bash
cd Eapp && npm run build:win
```

3. 复制 dist/backend 到 Eapp/build/win-unpacked/resources 目录



### 当前目录 HTTP 接口清单（基于 `routes.go`）

- **服务基址**: `http://localhost:18060`

```text
文件：routes.go
```

#### 页面与基础接口

| 方法 | 路径 | 说明 | 处理函数 |
|---|---|---|---|
| GET | `/` | 主页面（嵌入的 `XhsMcpWeb.html`） | 内联处理，读取 `webContent` |
| GET | `/login.html` | 登录页面 | 内联处理，读取 `webContent` |
| GET | `/health` | 健康检查 | `healthHandler` |

#### MCP（Streamable HTTP）

| 方法 | 路径 | 说明 | 处理函数 |
|---|---|---|---|
| ANY | `/mcp` | MCP 主端点 | `appServer.StreamableHTTPHandler()` |
| ANY | `/mcp/*path` | MCP 子路径代理 | `appServer.StreamableHTTPHandler()` |

#### REST API v1（前缀：`/api/v1`）

| 方法 | 路径 | 说明 | 处理函数 |
|---|---|---|---|
| GET | `/api/v1/login/status` | 检查登录状态 | `appServer.checkLoginStatusHandler` |
| POST | `/api/v1/login` | 登录 | `appServer.loginHandler` |
| GET | `/api/v1/sessions` | 列出会话 | `appServer.listSessionsHandler` |
| POST | `/api/v1/publish` | 发布内容 | `appServer.publishHandler` |
| GET | `/api/v1/feeds/list` | 获取笔记列表 | `appServer.listFeedsHandler` |
| GET | `/api/v1/feeds/search` | 搜索笔记 | `appServer.searchFeedsHandler` |
| GET | `/api/v1/browser/status` | 浏览器运行状态 | `appServer.browserStatusHandler` |
| POST | `/api/v1/browser/close` | 关闭一个浏览器 | `appServer.closeBrowserHandler` |
| POST | `/api/v1/browser/close-all` | 关闭所有浏览器 | `appServer.closeAllBrowsersHandler` |

### 使用提示

- 默认端口可通过参数修改：`xiaohongshu-mcp.exe -port 8080`
- 所有 API 基于 `gin`，返回 JSON；页面为内嵌 HTML 渲染。

- 变更摘要:
  - 生成了接口总览表，标注方法、路径、用途和处理函数，覆盖 `页面/健康检查/MCP/API v1` 全部端点。

