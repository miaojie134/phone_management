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
2.5 “办卡人已离职”风险处理逻辑
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
更新号码信息: 允许修改号码的状态、供应商、备注。当号码状态变更为“已注销”时，自动记录注销时间。

2.3 员工信息管理

功能描述:
创建员工: 保存新的员工记录，包括员工工号、姓名、部门，默认为在职状态。进行员工工号唯一性校验。
查询员工列表: 根据前端提供的筛选条件（在职状态）和搜索关键词（姓名、工号）以及分页参数，检索并返回员工列表。
查询员工详情: 根据员工 ID 获取单个员工的完整信息，包括其作为“办卡人”和“当前使用人”的号码列表。
更新员工信息: 允许修改员工的部门、在职状态。当员工状态从“在职”变更为“离职”时，记录离职日期，并触发关联的风险号码处理逻辑（见 2.5）。

2.4 号码分配与回收逻辑

功能描述:
分配号码:
校验目标号码是否为“闲置”状态，目标员工是否为“在职”状态。
更新号码记录，关联当前使用人员工 ID，将号码状态改为“在用”。
创建一条新的号码使用历史记录，记录使用开始时间。
回收号码:
校验目标号码是否为“在用”状态。
更新号码记录，清空当前使用人员工 ID，将号码状态改为“闲置”。
更新上一条与该号码和使用人相关的号码使用历史记录，记录使用结束时间。

2.5 “办卡人已离职”风险处理逻辑

功能描述:
当一个员工作为“办卡人”的员工被标记为“离职”时，系统自动执行以下操作：
查找该离职员工作为“办卡人”的所有手机号码中，状态仍为有效（非“已注销”）的记录。
对于这些被识别出的号码，如果其当前状态不是明确的风险提示状态，则将其状态自动更新为“待核实-办卡人离职”（或类似的风险标记状态），以便管理员后续跟进处理。

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
{
"token": "generated_jwt_token",
"user": {
"username": "admin_username",
"role": "admin"
}
}

响应 (401 Unauthorized): { "error": "无效的用户名或密码" }

POST /logout (需要认证)
描述: 管理员登出。
请求体: (无)

响应 (200 OK): { "message": "成功登出" }

3.3 手机号码 API (/api/v1/mobilenumbers) (均需要认证)

POST /
描述: 新增一个手机号码。
请求体:
{
"phoneNumber": "13800138000",
"applicantEmployeeId": "employee_db_id_or_business_key", // 办卡人员工 ID
"applicationDate": "2024-01-15",
"vendor": "中国移动",
"status": "闲置", // 初始状态
"remarks": "新购入卡"
}

响应 (201 Created): 返回创建成功的号码对象。

GET /
描述: 获取手机号码列表，支持分页、搜索和筛选。
查询参数:
page (可选, number, 默认 1): 页码。
limit (可选, number, 默认 10): 每页数量。
sortBy (可选, string): 排序字段。
sortOrder (可选, string, 'asc'或'desc'): 排序顺序。
search (可选, string): 搜索关键词 (匹配手机号、使用人、办卡人)。
status (可选, string): 号码状态筛选。
applicantStatus (可选, string, 'Active'或'Departed'): 办卡人当前在职状态筛选。

响应 (200 OK):
{
"data": [ /* 号码对象列表 */ ],
"pagination": {
"totalItems": 100,
"totalPages": 10,
"currentPage": 1,
"pageSize": 10
}
}

GET /:id
描述: 获取指定 ID 的手机号码详情。
路径参数: id (号码的数据库 ID)。

响应 (200 OK): 返回单个号码对象，包含其使用历史。

响应 (404 Not Found): { "error": "号码未找到" }

POST /:id/update (原 PUT /:id)
描述: 更新指定 ID 的手机号码信息 (主要用于更新状态、供应商、备注)。
路径参数: id。
请求体: (包含要更新的字段)
{
"status": "待注销",
"remarks": "用户申请停用",
"vendor": "中国联通"
}

响应 (200 OK): 返回更新后的号码对象。

POST /:id/assign
描述: 将指定 ID 的手机号码分配给一个员工。
路径参数: id (号码 ID)。
请求体:
{
"employeeId": "target_employee_db_id", // 目标使用人员工 ID
"assignmentDate": "2024-05-15" // 分配日期
}

响应 (200 OK): 返回更新后的号码对象。

POST /:id/unassign
描述: 从当前使用人处回收指定 ID 的手机号码。
路径参数: id (号码 ID)。
请求体: (可选，可包含回收日期)
{
"reclaimDate": "2024-06-01" // 回收日期
}

响应 (200 OK): 返回更新后的号码对象。

3.4 员工 API (/api/v1/employees) (均需要认证)
POST /
描述: 新增一个员工。
请求体:
{
"employeeId": "EMP001", // 员工业务工号
"fullName": "张三",
"department": "销售部"
// employmentStatus 默认为 "Active"
}

响应 (201 Created): 返回创建成功的员工对象。

GET /
描述: 获取员工列表，支持分页、搜索和筛选。
查询参数:
page, limit, sortBy, sortOrder (同号码列表)。
search (可选, string): 搜索关键词 (匹配姓名、工号)。
employmentStatus (可选, string, 'Active'或'Departed'): 在职状态筛选。

响应 (200 OK): 返回员工对象列表及分页信息。

GET /:id
描述: 获取指定 ID 的员工详情。
路径参数: id (员工的数据库 ID 或 业务工号 employeeId，需统一)。
响应 (200 OK): 返回单个员工对象，包含其作为“办卡人”和“当前使用人”的号码简要列表。

POST /:id/update (原 PUT /:id)
描述: 更新指定 ID 的员工信息。
路径参数: id。
请求体: (包含要更新的字段)
{
"department": "市场部",
"employmentStatus": "Departed", // 若改为 Departed
"terminationDate": "2024-05-10" // 离职日期
}

响应 (200 OK): 返回更新后的员工对象。

3.5 数据导入 API (/api/v1/import) (均需要认证)

POST /employees
描述: 批量导入员工数据。
请求体: multipart/form-data，包含一个名为 file 的 Excel 或 CSV 文件。

响应 (200 OK):
{
"message": "员工数据导入处理完成。",
"successCount": 95,
"errorCount": 5,
"errors": [
{ "row": 10, "employeeId": "EMP010", "reason": "员工工号已存在" },
{ "row": 25, "reason": "姓名不能为空" }
]
}

POST /mobilenumbers
描述: 批量导入手机号码数据。
请求体: multipart/form-data，包含一个名为 file 的 Excel 或 CSV 文件。

响应 (200 OK): 结构类似员工导入的响应，包含成功/失败统计及错误详情。

4. 数据模型详述

Users (管理员用户表)

id (PK, SERIAL 或 INT AUTO_INCREMENT, NOT NULL) - 主键 ID
username (VARCHAR(255), UNIQUE, NOT NULL) - 用户名
passwordHash (VARCHAR(255), NOT NULL) - 加密后的密码
role (VARCHAR(50), NOT NULL, DEFAULT 'admin') - 角色 (MVP 阶段固定为 'admin')
createdAt (TIMESTAMP, NOT NULL, DEFAULT CURRENT_TIMESTAMP) - 创建时间
updatedAt (TIMESTAMP, NOT NULL, DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP) - 更新时间

Employees (员工表)
id (PK, SERIAL 或 INT AUTO_INCREMENT, NOT NULL) - 主键 ID
employeeId (VARCHAR(100), UNIQUE, NOT NULL) - 员工业务工号
fullName (VARCHAR(255), NOT NULL) - 姓名
department (VARCHAR(255), NULL) - 部门
employmentStatus (VARCHAR(50), NOT NULL, DEFAULT 'Active') - 在职状态 (例如: 'Active', 'Departed')
hireDate (DATE, NULL) - 入职日期
terminationDate (DATE, NULL) - 离职日期
createdAt (TIMESTAMP, NOT NULL, DEFAULT CURRENT_TIMESTAMP)
updatedAt (TIMESTAMP, NOT NULL, DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)

MobileNumbers (手机号码表)
id (PK, SERIAL 或 INT AUTO_INCREMENT, NOT NULL) - 主键 ID
phoneNumber (VARCHAR(50), UNIQUE, NOT NULL) - 手机号码
applicantEmployeeDbId (FK, INT, NOT NULL) - 办卡人员工记录的数据库 ID (关联 Employees.id)
applicationDate (DATE, NOT NULL) - 办卡日期
currentEmployeeDbId (FK, INT, NULL) - 当前使用人员工记录的数据库 ID (关联 Employees.id)
status (VARCHAR(50), NOT NULL, DEFAULT '闲置') - 号码状态 (例如: '闲置', '在用', '待注销', '已注销', '待核实-办卡人离职')
vendor (VARCHAR(100), NULL) - 供应商
remarks (TEXT, NULL) - 备注
cancellationDate (DATE, NULL) - 注销日期
createdAt (TIMESTAMP, NOT NULL, DEFAULT CURRENT_TIMESTAMP)
updatedAt (TIMESTAMP, NOT NULL, DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)

NumberUsageHistory (号码使用历史表)
id (PK, SERIAL 或 INT AUTO_INCREMENT, NOT NULL) - 主键 ID
mobileNumberDbId (FK, INT, NOT NULL) - 手机号码记录的数据库 ID (关联 MobileNumbers.id)
employeeDbId (FK, INT, NOT NULL) - 使用人员工记录的数据库 ID (关联 Employees.id)
startDate (TIMESTAMP, NOT NULL) - 使用开始日期时间
endDate (TIMESTAMP, NULL) - 使用结束日期时间
createdAt (TIMESTAMP, NOT NULL, DEFAULT CURRENT_TIMESTAMP)
updatedAt (TIMESTAMP, NOT NULL, DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)

5. 关键业务逻辑点

唯一性校验: MobileNumbers.phoneNumber 和 Employees.employeeId 必须保证唯一。

状态流转: 后端需严格控制号码状态和员工在职状态的有效流转。

办卡人离职处理: 在员工被标记为离职时，后端自动触发对其作为“办卡人”的有效号码进行状态更新或标记的逻辑。

使用历史记录: 在号码分配和回收时，准确创建和更新 NumberUsageHistory 表，记录使用起止时间。

数据一致性: 在涉及多表操作时（如分配号码），应考虑事务处理以保证数据一致性。

外键约束: 数据库层面应建立正确的外键约束，以保证数据引用的完整性。

6. 安全性考量

输入校验: 对所有来自客户端的输入进行严格校验，防止 SQL 注入、XSS 等攻击。

密码存储: 管理员密码必须使用强哈希算法（如 bcrypt）加盐存储。

API 认证: 确保所有敏感 API 都受到 JWT 或类似机制的保护。

错误处理: 避免在错误信息中泄露过多敏感的系统内部细节。

依赖库安全: 定期更新所使用的第三方库，防范已知漏洞。
