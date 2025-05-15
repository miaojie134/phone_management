# 企业手机号码管理系统 MVP

企业手机号码管理系统后端 MVP 版本。

## 技术栈

- Go
- Gin Web Framework
- SQLite

## 目录结构说明

(后续可以补充详细的目录结构说明)

## 如何运行

1.  初始化 Go 模块: `go mod init your_project_module_name` (请替换 `your_project_module_name`)
2.  下载依赖: `go mod tidy`
3.  运行: `go run cmd/server/main.go`

## 环境变量配置

在运行应用程序之前，您可以设置以下环境变量来自定义其行为：

- `JWT_SECRET_KEY`: 用于签名和验证 JWT（JSON Web Tokens）的密钥。如果未设置，应用程序将使用一个默认密钥并打印警告（不推荐用于生产环境）。

  ```bash
  export JWT_SECRET_KEY="your-super-secure-and-long-secret-key"
  ```

- `SERVER_PORT`: 应用程序监听的端口号。如果未设置，将默认为 `8080`。
  ```bash
  export SERVER_PORT="8888"
  ```

您可以在启动应用程序的 shell 会话中直接设置这些变量，或者将它们添加到您的 shell 配置文件（如 `.bashrc`, `.zshrc`）中，或者使用 `.env` 文件配合像 `godotenv` 这样的库（如果项目后续引入）。
