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

- `SMTP_HOST`: SMTP 服务器的主机名或 IP 地址。
- `SMTP_PORT`: SMTP 服务器的端口号 (例如 587 或 465)。
- `SMTP_USERNAME`: (可选) 用于 SMTP 服务器认证的用户名。
- `SMTP_PASSWORD`: (可选) 用于 SMTP 服务器认证的密码。
- `SMTP_SENDER_EMAIL`: 发送邮件时使用的发件人邮箱地址。

  ```bash
  export SMTP_HOST="smtp.qiye.aliyun.com"
  export SMTP_PORT="465"
  export SMTP_USERNAME="miaojie@knowbox.cn"
  export SMTP_PASSWORD="9A2MACuWUIFV2QkzA"
  export SMTP_SENDER_EMAIL="miaojie@knowbox.cn"

  export TEST_RECIPIENT_EMAIL="guanxiao@knowbox.cn"
  export TEST_EMPLOYEE_NAME="管潇"
  export TEST_VERIFICATION_LINK= ""
  ```

您可以在启动应用程序的 shell 会话中直接设置这些变量，或者将它们添加到您的 shell 配置文件（如 `.bashrc`, `.zshrc`）中，或者使用 `.env` 文件配合像 `godotenv` 这样的库（如果项目后续引入）。

代理环境变量配置
export https_proxy="http://127.0.0.1:12334"

更新 swagger 文档
$(go env GOPATH)/bin/swag init -g cmd/server/main.go
