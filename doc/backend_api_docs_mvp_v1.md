企业手机号码管理系统 MVP - 后端功能及 API 接口文档

版本： 1.1
日期： 2025 年 5 月 15 日

目录

引言
1.1 目标
1.2 技术栈假设

核心后端功能模块
2.1 用户认证与授权
2.2 手机号码管理
2.3 员工信息管理
2.4 号码分配与回收逻辑
2.5 "办卡人已离职"风险处理逻辑
2.6 数据导入处理

API 接口设计
3.1 通用约定
3.2 认证 API (/api/v1/auth)
3.3 手机号码 API (/api/v1/mobilenumbers)
3.4 员工 API (/api/v1/employees)
3.5 数据导入 API (/api/v1/import)

数据模型详述

关键业务逻辑点

安全性考量

1. 引言 (Introduction)

1.1 目标:
本文档旨在详细描述企业手机号码管理系统 MVP 版本的后端核心功能和 API 接口设计，为后端开发提供清晰的指引。

1.2 技术栈假设:

后端框架: golang/gin

数据库: sqlite

认证机制: JWT (JSON Web Tokens)

2. 核心后端功能模块

2.1 用户认证与授权

功能描述:
验证管理员登录凭证的有效性。
为成功登录的管理员生成并下发身份令牌 (Token)。
校验后续 API 请求中携带的 Token，确保请求的合法性。
MVP 阶段仅支持单一管理员角色，无需复杂权限管理。
支持管理员登出，使当前 Token 失效。

2.2 手机号码管理

功能描述:
创建号码: 保存新的手机号码记录，包括手机号、办卡人、办卡日期、供应商、初始状态、备注。进行手机号码唯一性校验。
查询号码列表: 根据前端提供的筛选条件（号码状态、办卡人状态）和搜索关键词（手机号、使用人、办卡人）以及分页参数，从数据库中检索并返回号码列表。需要关联员工表获取办卡人及使用人姓名、办卡人当前在职状态。
查询号码详情: 根据号码 ID 获取单个号码的完整信息，包括其使用历史记录。
更新号码信息: 允许修改号码的状态、供应商、备注。当号码状态变更为"已注销"时，自动记录注销时间。

2.3 员工信息管理

功能描述:
创建员工: 保存新的员工记录，包括员工工号、姓名、部门，默认为在职状态。进行员工工号唯一性校验。
查询员工列表: 根据前端提供的筛选条件（在职状态）和搜索关键词（姓名、工号）以及分页参数，检索并返回员工列表。
查询员工详情: 根据员工 ID 获取单个员工的完整信息，包括其作为"办卡人"和"当前使用人"的号码列表。
更新员工信息: 允许修改员工的部门、在职状态。当员工状态从"在职"变更为"离职"时，记录离职日期，并触发关联的风险号码处理逻辑（见 2.5）。

2.4 号码分配与回收逻辑

功能描述:
分配号码:
校验目标号码是否为 `idle`（闲置）状态，目标员工是否为"在职"状态。
更新号码记录，关联当前使用人员工 ID，将号码状态改为 `in_use`（使用中）。
创建一条新的号码使用历史记录，记录使用开始时间。
回收号码:
校验目标号码是否为 `in_use`（使用中）状态。
更新号码记录，清空当前使用人员工 ID，将号码状态改为 `idle`（闲置）。
更新上一条与该号码和使用人相关的号码使用历史记录，记录使用结束时间。

2.5 "办卡人已离职"风险处理逻辑

功能描述:
当一个员工作为"办卡人"的员工被标记为"离职"时，系统自动执行以下操作：
查找该离职员工作为"办卡人"的所有手机号码中，状态仍为有效（非 `deactivated` 已注销）的记录。
对于这些被识别出的号码，如果其当前状态不是明确的风险提示状态，则将其状态自动更新为 `risk_pending`（待核实-办卡人离职），以便管理员后续跟进处理。

2.6 数据导入处理

功能描述:

员工数据导入:
*接收前端上传的员工数据文件（Excel/CSV）。
*解析文件内容。
*对每条数据进行格式校验和逻辑校验（如工号是否已存在）。
*将校验通过的数据批量存入员工数据库表。 \*返回导入操作的结果统计（成功数、失败数）及详细的错误信息列表。

号码数据导入:
*接收前端上传的号码数据文件（Excel/CSV）。
*解析文件内容。
*对每条数据进行格式校验和逻辑校验（如手机号格式、办卡人姓名/工号能否在员工库中匹配到对应的员工记录）。
*将校验通过的数据批量存入号码数据库表，并建立与办卡人的关联。 \*返回导入操作的结果统计及详细的错误信息列表。

2.7 号码使用确认流程 (新)
功能描述:
发起确认流程: 允许管理员为全部或特定员工群体发起手机号码使用情况的确认流程。
生成并分发令牌: 为每个需要确认的员工（或其名下的每个号码，根据设计选择）生成唯一的、有时效性的、安全的验证令牌。
发送通知邮件: 系统通过邮件将包含专属验证链接（内含令牌）的通知发送给相关员工。
验证令牌并展示信息: 用户点击链接后，系统验证令牌的有效性（是否存在、未过期、未使用），然后查询并展示该员工名下登记的所有手机号码。
处理用户反馈:
用户可以为名下每个号码选择"确认使用"或"报告问题"。
系统记录用户的反馈。对于"确认使用"的号码，可更新其最后确认日期；对于"报告问题"的号码，系统应标记并通知管理员进行跟进。
用户可以提交他们正在使用但未在系统名下登记的号码信息。
令牌状态管理: 令牌在使用后或过期后应标记为无效，防止重复使用。
管理员跟进: 系统提供界面或报告，供管理员查看确认进度、用户报告的问题以及用户新增的未登记号码，以便进行后续处理。

3. API 接口设计

3.1 通用约定

Base URL: /api/v1
请求格式: JSON
响应格式: JSON
认证: 除登录接口外，所有 API 均需在请求头中携带有效的 JWT Authorization: Bearer <token>。

HTTP 方法: 仅使用 GET 和 POST。更新操作使用 POST。

错误处理:
400 Bad Request: 请求参数错误或数据校验失败。响应体包含 { "error": "描述信息", "details": { ... } }。
401 Unauthorized: 未认证或 Token 无效/过期。
403 Forbidden: 已认证但无权限访问该资源 (MVP 阶段较少见)。
404 Not Found: 请求的资源不存在。
500 Internal Server Error: 服务器内部错误。

日期格式: YYYY-MM-DD 或 YYYY-MM-DDTHH:mm:ssZ (ISO 8601)。

3.2 认证 API (/api/v1/auth)

POST /login
描述: 管理员登录。
请求体:
{
"username": "admin_username",
"password": "admin_password"
}

响应 (200 OK):
JWT 详细信息:

- 过期时间: 24 小时
- Claims:
  - `jti` (JWT ID): UUID
  - `sub` (Subject): 用户名
  - `exp` (Expiration Time): 过期时间戳
  - `iss` (Issuer): "phone_system"
  - `aud` (Audience): ["admin"]
    {
    "token": "generated_jwt_token",
    "user": {
    "username": "admin_username",
    "role": "admin"
    }
    }

响应 (401 Unauthorized): { "error": "无效的用户名或密码" }
响应 (500 Internal Server Error): { "error": "无法生成 Token" } (来自代码实现)

POST /logout (需要认证)
描述: 管理员登出。此接口会将当前 Token 的 JTI (JWT ID) 添加到服务器端的拒绝列表中，使其立即失效。
请求体: (无)

响应 (200 OK): { "message": "成功登出" }
响应 (400 Bad Request): { "error": "登出上下文错误", "details": "JTI 或 EXP 未在上下文中找到或无效" } (来自代码实现)

3.3 手机号码 API (/api/v1/mobilenumbers) (均需要认证)

POST /
描述: 新增一个手机号码。
请求体:
{
"phoneNumber": "13800138000", // 手机号码 (必填, 11 位数字)
"applicantEmployeeId": "EMP001", // 办卡人员工业务工号 (必填, string)
"applicationDate": "2024-01-15", // 办卡日期 (必填, YYYY-MM-DD)
"status": "idle", // 初始状态 (可选, 枚举值见下, 默认为 idle)
"purpose": "办公用", // 号码用途 (可选, string, max 255)
"vendor": "中国移动", // 供应商 (可选, string, max 100)
"remarks": "新购入卡" // 备注 (可选, string, max 255)
}

`status` 枚举值:

- `idle` (闲置)
- `in_use` (使用中)
- `pending_deactivation` (待注销)
- `deactivated` (已注销)
- `risk_pending` (待核实-办卡人离职)
- `user_reported` (待核实-用户报告)

响应 (201 Created): 返回创建成功的号码对象 (`models.MobileNumber`)。

响应 (400 Bad Request): { "error": "请求参数错误或数据校验失败", "details": "..." } (例如日期格式错误，或手机号格式错误)
响应 (404 Not Found): { "error": "办卡人员工工号未找到", "details": "employeeId: EMP001" } (来自代码实现)
响应 (409 Conflict): { "error": "手机号码已存在" } (来自代码实现)
响应 (500 Internal Server Error): { "error": "创建手机号码失败", "details": "..." }

GET /
描述: 获取手机号码列表，支持分页、搜索和筛选。
查询参数:
page (可选, number, 默认 1): 页码。
limit (可选, number, 默认 10): 每页数量。
sortBy (可选, string): 排序字段 (例如: phoneNumber, applicationDate, status)。
sortOrder (可选, string, 'asc'或'desc', 默认 'asc'): 排序顺序。
search (可选, string): 搜索关键词 (匹配手机号、使用人姓名/工号、办卡人姓名/工号)。
status (可选, string): 号码状态筛选 (可选值: idle, in_use, pending_deactivation, deactivated, risk_pending, user_reported)。
applicantStatus (可选, string, 'Active'或'Departed'): 办卡人当前在职状态筛选。

响应 (200 OK):
{
"data": {
"items": [ /* MobileNumberResponse 对象列表 */ ],
"pagination": {
"totalItems": 100,
"totalPages": 10,
"currentPage": 1,
"pageSize": 10
}
},
"message": "手机号码列表获取成功"
}

`MobileNumberResponse` 对象结构 (部分字段，详见模型定义):
{
"id": 123,
"phoneNumber": "13800138000",
"applicantEmployeeId": "EMP001",
"applicantName": "张三",
"applicantStatus": "Active",
"applicationDate": "2023-01-15T00:00:00Z",
"currentEmployeeId": "EMP002",
"currentUserName": "李四",
"status": "in_use",
"purpose": "办公用",
"vendor": "中国移动",
"remarks": "备注信息",
"cancellationDate": null,
"createdAt": "2023-01-15T10:00:00Z",
"updatedAt": "2023-01-16T11:00:00Z",
"usageHistory": [ /* NumberUsageHistory 对象列表 */ ]
}

GET /mobilenumbers/{phoneNumber}
描述: 获取指定手机号码的详情。
路径参数:
phoneNumber (string, required): 手机号码字符串。

响应 (200 OK): 返回单个 `MobileNumberResponse` 对象 (结构同上列表接口)，包含其使用历史。

响应 (404 Not Found): { "error": "手机号码未找到" }
响应 (500 Internal Server Error): { "error": "获取手机号码详情失败", "details": "..." }

POST /mobilenumbers/{phoneNumber}/update
描述: 更新指定手机号码的信息 (主要用于更新状态、用途、供应商、备注)。当号码状态变更为"已注销"时，自动记录注销时间。至少需要提供一个有效的更新字段。

重要业务规则约束:

- 不能直接将状态更新为 `in_use`，必须通过分配操作 (POST /assign)
- 处于 `in_use` 状态的号码不能直接更新状态，必须先通过回收操作 (POST /unassign)

路径参数:
phoneNumber (string, required): 手机号码字符串。
请求体: (`MobileNumberUpdatePayload`)
{
"status": "pending_deactivation", // (可选, 枚举值见创建接口)
"purpose": "客户联系", // (可选, string, max 255)
"remarks": "用户申请停用", // (可选, string, max 255)
"vendor": "中国联通" // (可选, string, max 100)
}

响应 (200 OK): 返回更新后的 `MobileNumber` 对象。

响应 (400 Bad Request): { "error": "请求参数错误或数据校验失败 / 没有提供任何有效的更新字段 / 无效的手机号码格式", "details": "..." }
响应 (404 Not Found): { "error": "手机号码未找到" }
响应 (500 Internal Server Error): { "error": "更新手机号码失败", "details": "..." }

POST /mobilenumbers/{phoneNumber}/assign
描述: 将指定手机号码分配给一个员工。
路径参数:
phoneNumber (string, required): 手机号码字符串。
请求体: (`MobileNumberAssignPayload`)
{
"employeeId": "EMP002", // 目标使用人员工业务工号 (必填, string)
"assignmentDate": "2024-05-15", // 分配日期 (必填, YYYY-MM-DD)
"purpose": "日常办公" // 号码用途 (必填, string, max 255)
}

业务逻辑校验:

- 校验目标号码是否为 `idle` 状态
- 校验目标员工是否为"在职"状态
- 更新号码状态为 `in_use`，关联当前使用人员工 ID
- 创建一条新的号码使用历史记录

响应 (200 OK): 返回更新后的 `MobileNumber` 对象。

响应 (400 Bad Request): { "error": "请求参数错误 / 无效的日期格式 / 无效的手机号码格式", "details": "..." }
响应 (404 Not Found): { "error": "手机号码或目标员工工号未找到", "details": "..." }
响应 (409 Conflict): { "error": "操作冲突 (例如：号码非闲置，员工非在职)", "details": "..." }
响应 (500 Internal Server Error): { "error": "分配手机号码失败", "details": "..." }

POST /mobilenumbers/{phoneNumber}/unassign
描述: 从当前使用人处回收指定手机号码。
路径参数:
phoneNumber (string, required): 手机号码字符串。
请求体: (可选, `MobileNumberUnassignPayload`)
{
"reclaimDate": "2024-06-01" // 回收日期 (可选, YYYY-MM-DD, 默认当前时间)
}

业务逻辑校验:

- 校验目标号码是否为 `in_use` 状态
- 更新号码状态为 `idle`，清空当前使用人员工 ID
- 更新对应的号码使用历史记录，记录使用结束时间

响应 (200 OK): 返回更新后的 `MobileNumber` 对象。

响应 (400 Bad Request): { "error": "请求参数错误 / 无效的日期格式 / 无效的手机号码格式", "details": "..." }
响应 (404 Not Found): { "error": "手机号码未找到" }
响应 (409 Conflict): { "error": "操作冲突 (例如：号码非在用状态，或未找到有效的分配记录)", "details": "..." }
响应 (500 Internal Server Error): { "error": "回收手机号码失败", "details": "..." }

3.4 员工 API (/api/v1/employees) (均需要认证)
POST /
描述: 新增一个员工。员工业务工号 (employeeId) 由系统后端自动生成和分配。
请求体: (`CreateEmployeePayload`)
{
"fullName": "张三", // 姓名 (必填, string, max 255)
"phoneNumber": "13912345678", // 手机号 (可选, 11 位数字, 唯一)
"email": "zhangsan@example.com", // 邮箱 (可选, email 格式, 唯一, max 255)
"department": "销售部" // 部门 (可选, string, max 255)
}
注: `employmentStatus` 默认为 "Active"，由后端处理。

响应 (201 Created): 返回创建成功的 `Employee` 对象 (包含系统生成的 `employeeId`)。

响应 (400 Bad Request): { "error": "请求参数错误或数据校验失败", "details": "..." } (例如手机号或邮箱格式错误)
响应 (409 Conflict): { "error": "手机号码已存在 / 邮箱已存在 / 员工工号已存在 (理论上后端生成不会冲突)", "details": "..." }
响应 (500 Internal Server Error): { "error": "创建员工失败", "details": "..." }

GET /
描述: 获取员工列表，支持分页、搜索和筛选。
查询参数:
page, limit (同号码列表)。
sortBy (可选, string): 排序字段 (例如: employeeId, fullName, createdAt)。
sortOrder (可选, string, 'asc'或'desc', 默认 'desc'): 排序顺序。
search (可选, string): 搜索关键词 (匹配姓名、工号、手机号、邮箱)。
employmentStatus (可选, string, 'Active'或'Departed'): 在职状态筛选。

响应 (200 OK): 返回 `PagedEmployeesData` 对象，结构类似号码列表的响应，包含员工对象列表 (`Employee`) 及分页信息。
{
"data": {
"items": [ /* Employee 对象列表 */ ],
"pagination": { ... }
},
"message": "员工列表获取成功"
}

`Employee` 对象结构 (部分字段，详见模型定义):
{
"id": 1,
"employeeId": "EMP0000001",
"fullName": "张三",
"phoneNumber": "13912345678",
"email": "zhangsan@example.com",
"department": "销售部",
"employmentStatus": "Active",
"hireDate": "2023-01-01T00:00:00Z",
"terminationDate": null,
"createdAt": "2023-01-01T10:00:00Z",
"updatedAt": "2023-01-01T10:00:00Z"
}

GET /employees/{employeeId}
描述: 获取指定业务工号的员工详情。
路径参数:
employeeId (string, required): 员工业务工号。
响应 (200 OK): 返回单个 `EmployeeDetailResponse` 对象。
`EmployeeDetailResponse` 结构 (部分字段，详见模型定义):
{
"id": 1,
"employeeId": "EMP0000001",
"fullName": "张三",
"department": "销售部",
"employmentStatus": "Active",
// ... 其他员工基本信息 ...
"handledMobileNumbers": [
{ "id": 10, "phoneNumber": "13800138000", "status": "in_use" }
],
"usingMobileNumbers": [
{ "id": 10, "phoneNumber": "13800138000", "status": "in_use" },
{ "id": 11, "phoneNumber": "13700137000", "status": "idle" }
]
}
响应 (404 Not Found): { "error": "员工未找到" }
响应 (500 Internal Server Error): { "error": "获取员工详情失败", "details": "..." }

POST /employees/{employeeId}/update
描述: 更新指定业务工号的员工信息。至少需要提供一个有效的更新字段。
路径参数:
employeeId (string, required): 员工业务工号。
请求体: (`UpdateEmployeePayload`)
{
"department": "市场部", // (可选, string, max 255)
"employmentStatus": "Departed", // (可选, string, 枚举值: 'Active', 'Inactive', 'Departed')
"terminationDate": "2024-05-10" // (可选, YYYY-MM-DD, 仅当 employmentStatus 为 'Departed' 时有效或一同提供)
}
业务逻辑校验:

- 如果提供了 `terminationDate`，`employmentStatus` 必须是 `Departed`。
- 如果 `employmentStatus` 更新为非 `Departed`，则 `terminationDate` 不应有值或应被清空。

响应 (200 OK): 返回更新后的 `Employee` 对象。

响应 (400 Bad Request): { "error": "请求参数错误或数据校验失败 / 至少需要提供一个更新字段 / 状态与离职日期组合无效", "details": "..." }
响应 (404 Not Found): { "error": "员工未找到" }
响应 (500 Internal Server Error): { "error": "更新员工信息失败", "details": "..." }

3.5 数据导入 API (均需要认证)

POST /api/v1/employees/import
描述: 批量导入员工数据。后端会生成并分配 `employeeId`。
请求体: multipart/form-data，包含一个名为 `file` 的 CSV 文件。
CSV 文件要求:

- 编码: GBK 或 UTF-8 (带或不带 BOM)。
- 表头 (必须按此顺序): `fullName,phoneNumber,email,department`
  - `fullName` (string, 必填)
  - `phoneNumber` (string, 可选, 11 位数字, 唯一)
  - `email` (string, 可选, email 格式, 唯一)
  - `department` (string, 可选)

响应 (200 OK): (`BatchImportResponse`)
{
"message": "员工数据导入处理完成。成功: 95, 失败: 5",
"successCount": 95,
"errorCount": 5,
"errors": [
{ "rowNumber": 10, "rowData": ["旧李四", "139...", "lisi@", ""], "reason": "邮箱格式无效" },
{ "rowNumber": 25, "rowData": ["", "..."], "reason": "fullName 不能为空" }
]
}

POST /api/v1/mobilenumbers/import
描述: 批量导入手机号码数据。
请求体: multipart/form-data，包含一个名为 `file` 的 CSV 文件。
CSV 文件要求:

- 编码: GBK 或 UTF-8 (带或不带 BOM)。
- 表头 (必须按此顺序): `phoneNumber,applicantName,applicationDate,vendor`
  - `phoneNumber` (string, 必填, 11 位数字, 唯一)
  - `applicantName` (string, 必填, 办卡人姓名，系统会尝试匹配员工库中的 `fullName` 以关联 `applicantEmployeeId`)
  - `applicationDate` (string, 必填, YYYY-MM-DD)
  - `vendor` (string, 可选)
    (号码的 `status` 默认为 `idle`，`purpose` 和 `remarks` 默认为空)

响应 (200 OK): (`BatchImportMobileNumbersResponse`)
{
"message": "手机号码数据导入处理完成。成功: 90, 失败: 10",
"successCount": 90,
"errorCount": 10,
"errors": [
{ "rowNumber": 5, "rowData": ["138...", "不存在的人", "2023-01-01", "移动"], "reason": "办卡人姓名 '不存在的人' 未在员工库中找到" },
{ "rowNumber": 15, "rowData": ["137...", "张三", "日期错误", ""], "reason": "applicationDate 日期格式无效" }
]
}

3.6 号码确认 API (/api/v1/verification) (新)

POST /api/v1/verification/initiate
描述: (异步) 管理员调用此接口后，系统创建一个批处理任务来为目标员工生成唯一的 `VerificationTokens` 记录，并异步调用邮件服务发送包含专属确认链接的邮件。接口立即返回批处理任务的 ID。
请求体: (`InitiateVerificationRequest`)\n`json\n{\n  \"scope\": \"all_users | department | employee_ids\", // 必填, \"department\" 或 \"employee_ids\" 时 scopeValues 必填\n  \"scopeValues\": [\"value1\", \"value2\"], // 可选，当 scope 为 \"department\" (部门名称数组) 或 \"employee_ids\" (员工业务工号数组) 时需要\n  \"durationDays\": 7 // 必填，令牌有效期天数 (例如 1-30)\n}\n`\n\n 成功响应 (202 Accepted): (`InitiateVerificationResponse`)\n`json\n{\n  \"status\": \"success\",\n  \"message\": \"号码确认流程已作为批处理任务启动。\",\n  \"data\": {\n    \"batchId\": \"c7b5ba2a-3b9c-4b8d-8f3a-8c7d6e5f4g3h\" // 批处理任务的唯一ID\n  }\n}\n`\n\n 错误响应:\n\n- `400 Bad Request`: 请求参数无效 (如 `scope` 无效, `durationDays` 超出范围, 或 `scopeValues` 在需要时未提供，或 scopeValues 内容无法正确处理)。\n- `500 Internal Server Error`: 服务器内部错误 (如创建批处理任务失败或在准备阶段发生错误)。\n\nGET /api/v1/verification/batch/{batchId}/status\n 描述: 获取指定号码确认批处理任务的当前状态、整体进度（包括已处理员工数、令牌生成情况、邮件发送统计：尝试数、成功数、失败数）以及详细的错误报告（例如邮件发送失败的原因）。\n 路径参数: \* `batchId` (string, required): 批处理任务的唯一 ID (UUID)。\n 成功响应 (200 OK): 返回 `models.VerificationBatchTask` 对象。\n`json\n{\n  \"status\": \"success\",\n  \"message\": \"成功获取批处理任务状态。\",\n  \"data\": {\n    \"id\": \"c7b5ba2a-3b9c-4b8d-8f3a-8c7d6e5f4g3h\",\n    \"status\": \"InProgress | Completed | CompletedWithErrors | Failed | Pending\",\n    \"totalEmployeesToProcess\": 150,\n    \"tokensGeneratedCount\": 100,\n    \"emailsAttemptedCount\": 100,\n    \"emailsSucceededCount\": 95,\n    \"emailsFailedCount\": 5,\n    \"errorSummary\": \"[{\\\"employeeId\\\":\\\"emp123\\\",\\\"employeeName\\\":\\\"张三\\\",\\\"emailAddress\\\":\\\"zhangsan@example.com\\\",\\\"reason\\\":\\\"邮箱不存在或已禁用\\\"}, ...]\", // JSON 字符串化的 EmailFailureDetail 数组\n    \"requestedScopeType\": \"department\",\n    \"requestedScopeValues\": \"[\\\"技术部\\\", \\\"研发部\\\"]\", // JSON 字符串化的数组\n    \"requestedDurationDays\": 7,\n    \"createdAt\": \"2024-05-16T10:00:00Z\",\n    \"updatedAt\": \"2024-05-16T10:05:00Z\"\n  }\n}\n`\n\n 错误响应:\n\n- `400 Bad Request`: `batchId` 格式无效 (例如不是有效的 UUID)。\n- `404 Not Found`: 指定的 `batchId` 未找到。\n- `500 Internal Server Error`: 服务器内部错误。

GET /api/v1/verification/info (无需 JWT 认证, 令牌本身即是认证)
描述: 用户点击邮件链接后，前端页面调用此接口获取该用户需确认的号码信息以及此前通过该令牌报告的未列出号码。
查询参数: token (string, required) - 从邮件链接中获取的专属令牌。

响应 (200 OK): 返回 `models.VerificationInfo` 对象。
{
"status": "success",
"message": "成功获取待确认号码信息",
"data": {
"employeeId": "EMP001", // 员工业务 ID
"employeeName": "张三",
"phoneNumbers": [
{ "id": 123, "phoneNumber": "13800138000", "department": "技术部", "purpose": "办公用", "status": "confirmed", "userComment": null },
{ "id": 124, "phoneNumber": "13900139000", "department": "技术部", "purpose": "客户联系", "status": "pending", "userComment": null }
],
"previouslyReportedUnlisted": [
{ "phoneNumber": "18611120634", "userComment": "交付给我", "purpose": "项目临时", "reportedAt": "2023-10-27T10:00:00Z" }
],
"expiresAt": "2023-11-03T10:00:00Z"
}
}

响应 (400 Bad Request): { "error": "请求参数无效", "details": "缺少 token 参数" }
响应 (403 Forbidden): { "error": "无效或已过期的链接。" }
响应 (500 Internal Server Error): { "error": "获取验证信息失败", "details": "..." }

POST /api/v1/verification/submit (无需 JWT 认证, 令牌本身即是认证)
描述: 用户提交其号码确认结果。
查询参数: token (string, required)

请求体: (`models.VerificationSubmission`)
{
"verifiedNumbers": [
{ "mobileNumberId": 123, "action": "confirm_usage", "purpose": "办公用", "userComment": "" },
{ "mobileNumberId": 124, "action": "report_issue", "purpose": "个人使用", "userComment": "这个号码我已经不用了，给李四了" }
],
"unlistedNumbersReported": [
{ "phoneNumber": "13700137000", "purpose": "临时项目", "userComment": "这个号码公司给的，我一直在用，但列表里没有" }
]
}
详细校验规则:

- `verifiedNumbers` 和 `unlistedNumbersReported` 不能同时为空。
- 对于 `verifiedNumbers` 内的每个条目:
  - `mobileNumberId` (uint, 必填)
  - `action` (string, 必填, 枚举: `confirm_usage`, `report_issue`)
  - `purpose` (string, 必填, max 255)
  - `userComment` (string, 当 action 为 `report_issue` 时必填, max 500)
- 对于 `unlistedNumbersReported` 内的每个条目:
  - `phoneNumber` (string, 必填, 11 位数字)
  - `purpose` (string, 必填, max 255)
  - `userComment` (string, 可选, max 500)

响应 (200 OK): { "status": "success", "message": "您的反馈已成功提交，感谢您的配合！", "data": null }

响应 (400 Bad Request): { "error": "请求参数无效", "details": "... (例如 token 缺失, body 为空或字段校验失败)" }
响应 (403 Forbidden): { "error": "无效或已过期的链接，或已提交过。" }
响应 (500 Internal Server Error): { "error": "提交确认结果失败", "details": "..." }

GET /api/v1/verification/admin/phone-status (需要管理员认证)
描述: 管理员查看基于手机号码维度的确认流程状态和结果。
查询参数 (可选):
employee_id (string): 员工业务工号，用于筛选。
department (string): 部门名称，用于筛选。

响应 (200 OK): 返回 `PhoneVerificationStatusResponse` 对象。
{
"status": "success",
"message": "获取基于手机号码维度的确认流程状态成功",
"data": {
"summary": {
"totalPhonesCount": 200, // 系统中可用手机号码总数量（排除已注销）
"confirmedPhonesCount": 150, // 已确认使用的手机号码数
"reportedIssuesCount": 10, // 有问题的手机号码数 (用户报告)
"pendingPhonesCount": 40, // 待确认的手机号码数 (未响应或令牌未过期)
"newlyReportedPhonesCount": 5 // 新上报的未列出号码数
},
"confirmedPhones": [
{
"id": 123, "phoneNumber": "13800138000", "department": "技术部",
"currentUser": "张三 (EMP001)", "purpose": "办公用",
"confirmedBy": "张三 (EMP001)", "confirmedAt": "2024-05-17T10:00:00Z"
}
// ...更多已确认号码...
],
"pendingUsers": [
{ "employeeId": "EMP003", "fullName": "王五", "email": "wangwu@example.com", "expiresAt": "2024-05-20T23:59:59Z" }
// ...更多未响应用户...
],
"reportedIssues": [
{
"issueId": 1, "phoneNumber": "13900139001", "reportedBy": "李四 (EMP002)",
"comment": "此号码非我使用", "purpose": "项目专用", "originalStatus": "in_use",
"reportedAt": "2024-05-16T14:30:00Z", "adminActionStatus": "pending_review"
}
// ...更多报告的问题...
],
"unlistedNumbers": [
{ "phoneNumber": "13700137000", "reportedBy": "赵六 (EMP004)", "purpose": "备用", "userComment": "列表中没有", "reportedAt": "2024-05-15T09:00:00Z" }
// ...更多用户上报的未列出号码...
]
}
}

响应 (400 Bad Request): { "error": "请求参数错误", "details": "..." }
响应 (500 Internal Server Error): { "error": "获取确认流程状态失败", "details": "..." }

4. 数据模型详述

Users (管理员用户表)

id (PK, INT64 AUTO_INCREMENT, NOT NULL) - 主键 ID
username (VARCHAR(255), UNIQUE, NOT NULL) - 用户名
passwordHash (VARCHAR(255), NOT NULL) - 加密后的密码
role (VARCHAR(50), NOT NULL, DEFAULT 'admin') - 角色 (MVP 阶段固定为 'admin')
createdAt (TIMESTAMP, NOT NULL, autoCreateTime) - 创建时间 (由 GORM 自动管理)
updatedAt (TIMESTAMP, NOT NULL, autoUpdateTime) - 更新时间 (由 GORM 自动管理)
deletedAt (TIMESTAMP, NULL, INDEX) - 软删除时间 (由 GORM 自动管理)

Employees (员工表)
id (PK, INT64 AUTO_INCREMENT, NOT NULL) - 主键 ID
employeeId (VARCHAR(10), UNIQUE, NOT NULL) - 员工业务工号 (例如: EMP0000001, 由后端自动生成)
fullName (VARCHAR(255), NOT NULL) - 姓名
phoneNumber (VARCHAR(11), NULL, UNIQUE) - 员工手机号码 (11 位数字, 可选, 唯一)
email (VARCHAR(255), NULL, UNIQUE) - 员工邮箱 (可选, 唯一)
department (VARCHAR(255), NULL) - 部门
employmentStatus (VARCHAR(50), NOT NULL, DEFAULT 'Active') - 在职状态 (例如: 'Active', 'Departed', 'Inactive')
hireDate (DATE, NULL) - 入职日期
terminationDate (DATE, NULL) - 离职日期
createdAt (TIMESTAMP, NOT NULL, autoCreateTime) - 创建时间 (由 GORM 自动管理)
updatedAt (TIMESTAMP, NOT NULL, autoUpdateTime) - 更新时间 (由 GORM 自动管理)
deletedAt (TIMESTAMP, NULL, INDEX) - 软删除时间 (由 GORM 自动管理)

MobileNumbers (手机号码表)
id (PK, UINT AUTO_INCREMENT, NOT NULL) - 主键 ID
phoneNumber (VARCHAR(11), UNIQUE, NOT NULL) - 手机号码 (11 位数字)
applicantEmployeeId (VARCHAR(10), NOT NULL) - 办卡人员工业务工号 (关联 Employees.employeeId)
applicationDate (DATE, NOT NULL) - 办卡日期
currentEmployeeId (VARCHAR(10), NULL) - 当前使用人员工业务工号 (关联 Employees.employeeId)
status (VARCHAR(50), NOT NULL, DEFAULT 'idle') - 号码状态 (例如: 'idle', 'in_use', 'pending_deactivation', 'deactivated', 'risk_pending', 'user_reported')
purpose (VARCHAR(255), NULL) - 号码用途 (例如: '办公', '客户联系', '个人使用')
vendor (VARCHAR(100), NULL) - 供应商
remarks (TEXT, NULL) - 备注
cancellationDate (DATE, NULL) - 注销日期
lastConfirmationDate (TIMESTAMP, NULL) - 最后确认日期 (由号码确认流程更新)
createdAt (TIMESTAMP, NOT NULL, autoCreateTime) - 创建时间 (由 GORM 自动管理)
updatedAt (TIMESTAMP, NOT NULL, autoUpdateTime) - 更新时间 (由 GORM 自动管理)
deletedAt (TIMESTAMP, NULL, INDEX) - 软删除时间 (由 GORM 自动管理)

NumberUsageHistory (号码使用历史表)
id (PK, UINT AUTO_INCREMENT, NOT NULL) - 主键 ID
mobileNumberDbId (FK, UINT, NOT NULL) - 手机号码记录的数据库 ID (关联 MobileNumbers.id)
employeeId (VARCHAR(10), NOT NULL) - 使用人员工业务工号 (关联 Employees.employeeId)
startDate (TIMESTAMP, NOT NULL) - 使用开始日期时间
endDate (TIMESTAMP, NULL) - 使用结束日期时间
purpose (VARCHAR(255), NULL) - 本次分配的用途
createdAt (TIMESTAMP, NOT NULL, autoCreateTime) - 创建时间 (由 GORM 自动管理)
updatedAt (TIMESTAMP, NOT NULL, autoUpdateTime) - 更新时间 (由 GORM 自动管理)
deletedAt (TIMESTAMP, NULL, INDEX) - 软删除时间 (由 GORM 自动管理)

VerificationTokens (号码确认令牌表) (新)
id (PK, UINT AUTO_INCREMENT, NOT NULL) - 主键 ID
employeeId (VARCHAR(10), NOT NULL, INDEX) - 关联的员工业务工号 (参照 Employees.employeeId)
token (VARCHAR(255), UNIQUE, NOT NULL, INDEX) - 唯一验证令牌 (例如 UUID)
status (VARCHAR(50), NOT NULL, DEFAULT 'pending') - 令牌状态 (例如: 'pending', 'used', 'expired')
expiresAt (TIMESTAMP, NOT NULL) - 令牌过期时间
createdAt (TIMESTAMP, NOT NULL, autoCreateTime) - 创建时间 (由 GORM 自动管理)
updatedAt (TIMESTAMP, NOT NULL, autoUpdateTime) - 更新时间 (由 GORM 自动管理)
deletedAt (TIMESTAMP, NULL, INDEX) - 软删除时间 (由 GORM 自动管理)

VerificationSubmissionLog (号码确认提交日志表) (新)
描述: 记录用户通过号码确认流程提交的所有反馈，包括确认使用、报告问题和上报未列出号码。

id (PK, UINT AUTO_INCREMENT, NOT NULL) - 主键 ID
employeeId (VARCHAR(10), NOT NULL, INDEX) - 提交操作的员工业务工号 (参照 Employees.employeeId)
verificationTokenId (UINT, NOT NULL, INDEX) - 关联的验证令牌 ID (参照 VerificationTokens.id)
mobileNumberId (UINT, NULL, INDEX) - 关联的系统内手机号码 ID (如果操作针对现有号码, 参照 MobileNumbers.id)
phoneNumber (VARCHAR(20), NOT NULL, INDEX) - 操作涉及的手机号码 (对于系统内号码，此为冗余；对于未列出号码，此为主要标识)
actionType (VARCHAR(50), NOT NULL, INDEX) - 操作类型 (例如: 'confirm_usage', 'report_issue', 'report_unlisted')
purpose (VARCHAR(255), NULL) - 用户提供/确认的号码用途
userComment (TEXT, NULL) - 用户备注
createdAt (TIMESTAMP, NOT NULL, autoCreateTime) - 创建时间 (由 GORM 自动管理)
updatedAt (TIMESTAMP, NOT NULL, autoUpdateTime) - 更新时间 (由 GORM 自动管理)
deletedAt (TIMESTAMP, NULL, INDEX) - 软删除时间 (由 GORM 自动管理)

注：管理员跟进状态 (如 `adminActionStatus`, `adminRemarks`) 可通过此日志表派生并在服务层或单独的管理模块中进行管理。

VerificationBatchTasks (号码确认批处理任务表) (新)
描述: 存储号码确认流程的批处理任务信息。

id (VARCHAR(36), PK, NOT NULL) - 批处理任务的唯一 ID (UUID)
status (VARCHAR(50), NOT NULL, INDEX) - 任务状态 (例如: 'Pending', 'InProgress', 'Completed', 'CompletedWithErrors', 'Failed')
totalEmployeesToProcess (INT, NOT NULL) - 此批任务需要处理的总员工数
tokensGeneratedCount (INT, NOT NULL, DEFAULT 0) - 已成功生成令牌的员工数
emailsAttemptedCount (INT, NOT NULL, DEFAULT 0) - 尝试发送邮件的次数
emailsSucceededCount (INT, NOT NULL, DEFAULT 0) - 成功发送邮件的次数
emailsFailedCount (INT, NOT NULL, DEFAULT 0) - 发送邮件失败的次数
errorSummary (TEXT, NULL) - 错误概要 (例如，JSON 字符串化的 EmailFailureDetail 数组)
requestedScopeType (VARCHAR(50), NOT NULL) - 请求的范围类型 (例如: 'all_users', 'department', 'employee_ids')
requestedScopeValues (TEXT, NULL) - 请求的范围值 (例如，部门名称或员工 ID 的 JSON 数组字符串)
requestedDurationDays (INT, NOT NULL) - 请求的令牌有效期天数
createdAt (TIMESTAMP, NOT NULL, autoCreateTime) - 创建时间 (由 GORM 自动管理)
updatedAt (TIMESTAMP, NOT NULL, autoUpdateTime) - 更新时间 (由 GORM 自动管理)
deletedAt (TIMESTAMP, NULL, INDEX) - 软删除时间 (由 GORM 自动管理)

5. 关键业务逻辑点

唯一性校验: MobileNumbers.phoneNumber 和 Employees.employeeId 必须保证唯一。

状态流转约束:

- 后端需严格控制号码状态和员工在职状态的有效流转
- 手机号码状态约束：
  - 不能直接将状态更新为 `in_use`，必须通过分配操作
  - 处于 `in_use` 状态的号码不能直接更新状态，必须先进行回收操作
  - 状态值使用英文常量：`idle`, `in_use`, `pending_deactivation`, `deactivated`, `risk_pending`, `user_reported`

办卡人离职处理: 在员工被标记为离职时，后端自动触发对其作为"办卡人"的有效号码进行状态更新，将其标记为 `risk_pending` 状态。

使用历史记录: 在号码分配和回收时，准确创建和更新 NumberUsageHistory 表，记录使用起止时间。

数据一致性: 在涉及多表操作时（如分配号码），应考虑事务处理以保证数据一致性。

外键约束: 数据库层面应建立正确的外键约束，以保证数据引用的完整性。

6. 安全性考量

输入校验: 对所有来自客户端的输入进行严格校验，防止 SQL 注入、XSS 等攻击。

密码存储: 管理员密码必须使用强哈希算法（如 bcrypt）加盐存储。

API 认证: 确保所有敏感 API 都受到 JWT 或类似机制的保护。

错误处理: 避免在错误信息中泄露过多敏感的系统内部细节。

依赖库安全: 定期更新所使用的第三方库，防范已知漏洞。

7. 项目文件夹结构
   企业手机号码管理系统\_mvp/
   ├── cmd/
   │ └── server/
   │ └── main.go # 应用主入口，启动 HTTP 服务器
   ├── configs/ # (建议) 配置文件目录 (例如 config.yaml)
   ├── internal/
   │ ├── auth/ # (建议) 存放认证授权相关逻辑 (JWT 生成、校验、中间件等)
   │ ├── handlers/ # HTTP 请求处理器 (Controller 层)
   │ │ ├── auth_handler.go # 处理 /api/v1/auth 相关请求
   │ │ ├── employee_handler.go # 处理 /api/v1/employees 相关请求 (包含员工数据导入)
   │ │ ├── mobilenumber_handler.go # 处理 /api/v1/mobilenumbers 相关请求 (包含号码数据导入)
   │ │ ├── verification_handler.go # 处理 /api/v1/verification 相关请求
   │ │ └── common_types.go # API 层通用的请求/响应结构体
   │ ├── models/ # 数据模型 (对应数据库表和 API 请求/响应体)
   │ │ ├── user.go # 管理员用户模型 (Users)
   │ │ ├── employee.go # 员工模型 (Employees)
   │ │ ├── mobilenumber.go # 手机号码模型 (MobileNumbers)
   │ │ ├── number_usage_history.go # 号码使用历史模型 (NumberUsageHistory)
   │ │ ├── verification.go # 号码确认流程相关模型 (VerificationToken, VerificationBatchTask 等)
   │ │ └── verification_submission_log.go # 号码确认提交日志模型
   │ ├── repositories/ # (建议) 数据存储库层 (封装数据库操作)
   │ │ ├── user_repository.go
   │ │ ├── employee_repository.go
   │ │ ├── mobilenumber_repository.go
   │ │ ├── number_usage_history_repository.go
   │ │ └── verification_repository.go # (建议) 处理验证流程相关的数据存取
   │ ├── routes/ # API 路由定义
   │ │ └── router.go # 配置 Gin 路由，将 URL 映射到 handlers
   │ └── services/ # 业务逻辑服务层
   │ ├── employee_service.go # 员工管理服务 (包含员工数据导入逻辑)
   │ ├── mobilenumber_service.go # 手机号码管理服务 (包含号码数据导入逻辑)
   │ └── verification_service.go # 号码确认流程服务
   ├── migrations/ # (建议) 数据库迁移脚本
   │ └── 001_create_initial_tables.sql # 初始化数据库表的 SQL 脚本
   ├── pkg/
   │ ├── db/ # 数据库连接与操作相关
   │ │ └── sqlite.go # SQLite 数据库初始化及连接管理
   │ ├── logger/ # (建议) 日志封装
   │ │ └── logger.go
   │ └── utils/ # (建议) 通用工具函数
   │ └── response.go # 统一 API 响应格式等
   ├── go.mod # Go 模块依赖文件
   ├── go.sum # Go 模块校验和文件
   └── README.md # 项目说明文档
