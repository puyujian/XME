# GitHub Actions 自动构建和发布说明

## 功能说明

本工作流会自动执行以下步骤：
1. 编译 Go 后端程序为 Windows exe 文件
2. 打包 Electron 前端为 Windows 便携式应用
3. 创建 GitHub Release 并上传构建产物

## 触发条件

工作流会在以下情况自动触发：
- 推送代码到 `main` 分支
- 手动触发（通过 GitHub Actions 界面的 "Run workflow" 按钮）

## 构建产物

构建完成后，会生成：
- **Windows 便携式应用**：`小红书管理平台-{version}.exe`
  - 包含 Electron 前端界面
  - 内嵌 Go 后端服务
  - 无需安装，双击即可运行

## 发布方式

每次成功构建会自动创建一个新的 Release（预发布版本），包含：
- 标签名称：`build-{运行ID}`
- 构建编号和提交信息
- 可下载的 exe 文件

## 手动触发构建

1. 进入仓库的 Actions 页面
2. 选择 "Build and Release Windows Package" 工作流
3. 点击 "Run workflow" 按钮
4. 选择分支后点击 "Run workflow" 确认

## 注意事项

- 工作流需要 `contents: write` 权限来创建 Release
- 构建环境使用 `windows-latest` 以确保兼容性
- 构建过程约需 5-10 分钟
- 每次构建会创建一个新的预发布版本，可以在 Releases 页面管理

## 构建流程

```
1. Checkout 代码
2. 安装 Go 环境
3. 编译后端 -> dist/xiaohongshu-mcp.exe
4. 复制后端到 Eapp/backend/
5. 安装 Node.js 环境
6. 安装 npm 依赖
7. 运行 electron-builder
8. 创建 Release 并上传
```

## 本地测试

如果想在本地测试构建流程：

### Windows 系统：
```bash
# 1. 编译 Go 后端
go build -ldflags "-s -w" -o dist/xiaohongshu-mcp.exe .

# 2. 复制到 Eapp/backend
mkdir Eapp\backend
copy dist\xiaohongshu-mcp.exe Eapp\backend\

# 3. 安装依赖并打包
cd Eapp
npm install
npm run build:win
```

### Linux/Mac 系统：
```bash
# 1. 编译 Go 后端（交叉编译为 Windows）
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o dist/xiaohongshu-mcp.exe .

# 2. 复制到 Eapp/backend
mkdir -p Eapp/backend
cp dist/xiaohongshu-mcp.exe Eapp/backend/

# 3. 安装依赖并打包（需要 wine）
cd Eapp
npm install
npm run build:win
```

注意：在 Linux/Mac 上打包 Windows 应用需要安装 wine。
