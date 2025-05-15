精确的 AI 指令 (Prompts)

技术栈: Golang + Gin + SQLite + JWT

阶段 1: 项目初始化和基础设置

1.1 项目结构指令:

为‘企业手机号码管理系统 MVP’后端项目规划 Go 项目文件夹结构。技术栈：Golang、Gin Web 框架、SQLite。项目结构应包含 cmd/server, internal/handlers, internal/models, internal/services, internal/routes, pkg/db 等目录。请说明各目录用途，并参考 backend_api_docs_mvp_v1 文档的整体范围。

1.2 基础服务器和数据库连接指令:

基于规划的 Go 项目结构，在 cmd/server/main.go 中生成基础的 Gin 服务器设置代码，包含启动 HTTP 服务器的逻辑。同时，在 pkg/db 目录下生成使用 database/sql 包和 mattn/go-sqlite3 驱动连接 SQLite 数据库的配置、基础连接及初始化代码模块。数据库文件路径通过环境变量配置。

阶段 2: 数据模型定义

2.1 员工模型定义指令:

根据 backend_api_docs_mvp_v1 文档第 4 节‘数据模型详述’中关于员工(Employees)的描述，在 internal/models 目录下生成定义员工数据模型的 Go 结构体代码。包含所有字段，使用适当的 Go 类型和结构体标签（如 json 标签，考虑 SQLite 数据库）。

(用户提示：对每个核心数据模型重复类似的指令，例如 MobileNumbers, NumberUsageHistory, Users)

(用户可选指令) 2.2 数据库表创建 SQL 指令:

基于定义的 Go 数据模型 (Employees, MobileNumbers, NumberUsageHistory, Users)，为我生成用于在 SQLite 中创建相应表的 SQL DDL 语句。

阶段 3: 用户认证 API (文档 3.2 节)

3.1 登录 API 实现指令:

根据 backend_api_docs_mvp_v1 文档第 3.2 节 POST /api/v1/auth/login 的描述，在 internal/handlers 中生成管理员登录的 Gin 处理函数代码，并在 internal/routes 中设置相应路由。功能：从请求体绑定用户名和密码，与 SQLite 数据库中的 Users 表校验用户信息，成功后使用 golang-jwt/jwt (v4 或 v5) 库生成 JWT 并返回。包含使用 golang.org/x/crypto/bcrypt 进行密码哈希校验的逻辑。

3.2 登出 API 和 JWT 中间件实现指令:

实现 POST /api/v1/auth/logout 端点 (如 backend_api_docs_mvp_v1 文档 3.2 节所述，MVP 阶段返回成功消息即可)。同时，创建 Gin 中间件用于 JWT 认证，保护后续 API 路由。中间件需从请求头 Authorization: Bearer <token> 提取并使用 golang-jwt/jwt (v4 或 v5) 库验证 Token。

阶段 4: 核心资源 API - 以手机号码为例 (文档 3.3 节)

4.1 创建号码 API 实现指令 (POST /):

根据 backend_api_docs_mvp_v1 文档第 3.3 节 POST / (新增一个手机号码) 的描述，生成对应的 Gin 路由和处理函数代码。功能：从请求体绑定数据并验证（phoneNumber, applicantEmployeeDbId 必填），数据保存到 SQLite 的 MobileNumbers 表中，进行手机号码唯一性校验。此路由受 JWT 认证中间件保护。

4.2 获取号码列表 API 实现指令 (GET /):

实现 GET /api/v1/mobilenumbers 端点 (如 backend_api_docs_mvp_v1 文档 3.3 节所述)。Gin 处理函数需处理查询参数：page, limit, sortBy, sortOrder, search, status, applicantStatus。后端逻辑需正确处理参数，从 SQLite 查询数据，进行必要的表连接（如与 Employees 表）以获取办卡人和使用人姓名及办卡人状态。返回数据格式符合文档中的分页结构。

(用户提示：逐步为 GET /:id, POST /:id/update, POST /:id/assign, POST /:id/unassign 生成类似的指令)

阶段 5: 实现核心业务逻辑 (文档 2 节和 5 节)

5.1 办卡人离职风险处理逻辑实现指令:

在 POST /api/v1/employees/:id/update API 的业务逻辑中，集成‘办卡人已离职’风险处理逻辑。根据 backend_api_docs_mvp_v1 文档第 2.5 节描述，在 internal/services 或处理函数中实现此逻辑：如果员工被标记为离职，系统自动查找其作为办卡人的有效号码，并将其状态更新为‘待核实-办卡人离职’。请提供相关的 Go 代码片段或详细实现步骤。

阶段 6: 数据导入 (文档 3.5 节)

6.1 员工数据导入 API 实现指令:

实现员工数据批量导入功能 POST /api/v1/import/employees (如 backend_api_docs_mvp_v1 文档 3.5 节所述)。Gin 处理函数需接收 multipart/form-data 上传的 Excel 或 CSV 文件，使用如 github.com/xuri/excelize/v2 (Excel) 或 encoding/csv (CSV) 库解析文件，校验数据（如工号唯一性），批量存入 SQLite。返回包含成功/失败统计及错误详情的 JSON 响应。

(用户提示：为号码数据导入 POST /api/v1/import/mobilenumbers 生成类似指令)

阶段 7: 安全性和错误处理 (文档 3.1 节和 6 节)

7.1 Gin 错误处理机制实现指令:

为 Gin 应用设置错误处理机制或中间件，捕获处理函数返回的错误，并根据 backend_api_docs_mvp_v1 文档 3.1 节定义的错误响应格式（如 400, 401, 404, 500 错误）返回 JSON 响应。请展示如何在处理函数中返回自定义错误以便此中间件捕获。
