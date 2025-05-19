package handlers

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/phone_management/internal/models"
	"github.com/phone_management/internal/repositories"
	"github.com/phone_management/internal/services"
	"github.com/phone_management/pkg/utils"
)

// EmployeeHandler 封装了员工相关的 HTTP 处理逻辑
type EmployeeHandler struct {
	service services.EmployeeService
}

// NewEmployeeHandler 创建一个新的 EmployeeHandler 实例
func NewEmployeeHandler(service services.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{service: service}
}

// PagedEmployeesData 定义了员工列表的分页响应结构
type PagedEmployeesData struct {
	Items      []models.Employee `json:"items"`
	Pagination PaginationInfo    `json:"pagination"`
}

// BatchImportErrorDetail 描述了批量导入中单行数据的错误信息
type BatchImportErrorDetail struct {
	RowNumber int      `json:"rowNumber"`         // CSV中的原始行号 (从1开始计数，包括表头)
	RowData   []string `json:"rowData,omitempty"` // 可选，原始行数据
	Reason    string   `json:"reason"`            // 错误原因
}

// BatchImportResponse 定义了批量导入员工的响应结构
type BatchImportResponse struct {
	Message      string                   `json:"message"`
	SuccessCount int                      `json:"successCount"`
	ErrorCount   int                      `json:"errorCount"`
	Errors       []BatchImportErrorDetail `json:"errors,omitempty"`
}

// CreateEmployee godoc
// @Summary 新增一个员工
// @Description 从请求体绑定数据并验证，数据保存到数据库。员工工号需唯一。
// @Tags Employees
// @Accept json
// @Produce json
// @Param employee body CreateEmployeePayload true "员工信息"
// @Success 201 {object} utils.SuccessResponse{data=models.Employee} "创建成功的员工对象"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误或数据校验失败"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 409 {object} utils.APIErrorResponse "员工工号已存在"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /employees [post]
// @Security BearerAuth
func (h *EmployeeHandler) CreateEmployee(c *gin.Context) {
	var payload models.CreateEmployeePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	employeeToCreate := &models.Employee{
		FullName:    payload.FullName,
		PhoneNumber: payload.PhoneNumber,
		Email:       payload.Email,
		Department:  payload.Department,
	}

	createdEmployee, err := h.service.CreateEmployee(employeeToCreate)
	if err != nil {
		if errors.Is(err, repositories.ErrEmployeeIDExists) {
			utils.RespondConflictError(c, repositories.ErrEmployeeIDExists.Error())
		} else {
			utils.RespondInternalServerError(c, "创建员工失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusCreated, createdEmployee, "员工创建成功")
}

// GetEmployees godoc
// @Summary 获取员工列表
// @Description 根据查询参数获取员工列表，支持分页、搜索和筛选
// @Tags Employees
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Param sortBy query string false "排序字段 (例如: employeeId, fullName, createdAt)"
// @Param sortOrder query string false "排序顺序 ('asc'或'desc')" default("desc")
// @Param search query string false "搜索关键词 (匹配姓名、工号)"
// @Param employmentStatus query string false "在职状态筛选 ('Active'或'Departed')"
// @Success 200 {object} utils.SuccessResponse{data=PagedEmployeesData} "成功响应，包含员工列表和分页信息"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /employees [get]
// @Security BearerAuth
func (h *EmployeeHandler) GetEmployees(c *gin.Context) {
	type GetEmployeesQuery struct {
		Page             int    `form:"page,default=1"`
		Limit            int    `form:"limit,default=10"`
		SortBy           string `form:"sortBy"`
		SortOrder        string `form:"sortOrder,default=desc"` // 默认降序
		Search           string `form:"search"`
		EmploymentStatus string `form:"employmentStatus" binding:"omitempty,oneof=Active Departed"` // 校验可选值
	}

	var queryParams GetEmployeesQuery
	if err := c.ShouldBindQuery(&queryParams); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	if queryParams.SortOrder != "asc" && queryParams.SortOrder != "desc" {
		queryParams.SortOrder = "desc" // 确保是有效值，否则默认为 desc
	}
	if queryParams.Limit <= 0 {
		queryParams.Limit = 10
	}
	if queryParams.Page <= 0 {
		queryParams.Page = 1
	}

	employees, totalItems, err := h.service.GetEmployees(
		queryParams.Page,
		queryParams.Limit,
		queryParams.SortBy,
		queryParams.SortOrder,
		queryParams.Search,
		queryParams.EmploymentStatus,
	)

	if err != nil {
		utils.RespondInternalServerError(c, "获取员工列表失败", err.Error())
		return
	}

	totalPages := int64(0)
	if queryParams.Limit > 0 {
		totalPages = (totalItems + int64(queryParams.Limit) - 1) / int64(queryParams.Limit)
	}
	if totalPages == 0 && totalItems > 0 {
		totalPages = 1
	}

	pagedData := PagedEmployeesData{
		Items: employees,
		Pagination: PaginationInfo{ // 复用 mobilenumber_handler.go 中的 PaginationInfo
			TotalItems:  totalItems,
			TotalPages:  totalPages,
			CurrentPage: queryParams.Page,
			PageSize:    queryParams.Limit,
		},
	}

	utils.RespondSuccess(c, http.StatusOK, pagedData, "员工列表获取成功")
}

// GetEmployeeByID godoc
// @Summary 获取指定业务工号的员工详情
// @Description 根据路径参数员工业务工号获取单个员工的完整信息，包含其作为"办卡人"和"当前使用人"的号码简要列表。
// @Tags Employees
// @Accept json
// @Produce json
// @Param employeeId path string true "员工业务工号"
// @Success 200 {object} utils.SuccessResponse{data=models.EmployeeDetailResponse} "成功响应，包含员工详情"
// @Failure 400 {object} utils.APIErrorResponse "无效的员工工号格式 (保留，以防未来有格式校验)"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 404 {object} utils.APIErrorResponse "员工未找到"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /employees/{employeeId} [get]
// @Security BearerAuth
func (h *EmployeeHandler) GetEmployeeByID(c *gin.Context) {
	employeeIdStr := c.Param("employeeId") // 将从 :employeeId 路径参数获取

	// 调用新的服务层方法
	employeeDetail, err := h.service.GetEmployeeDetailByEmployeeID(employeeIdStr)
	if err != nil {
		if errors.Is(err, services.ErrEmployeeNotFound) {
			utils.RespondNotFoundError(c, "员工")
		} else {
			utils.RespondInternalServerError(c, "获取员工详情失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusOK, employeeDetail, "员工详情获取成功")
}

// UpdateEmployee godoc
// @Summary 更新指定业务工号的员工信息
// @Description 根据员工业务工号更新员工的部门、在职状态或离职日期。
// @Tags Employees
// @Accept json
// @Produce json
// @Param employeeId path string true "员工业务工号"
// @Param employeeUpdate body models.UpdateEmployeePayload true "要更新的员工字段"
// @Success 200 {object} utils.SuccessResponse{data=models.Employee} "更新后的员工对象"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误或数据校验失败"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 404 {object} utils.APIErrorResponse "员工未找到"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /employees/{employeeId}/update [post]
// @Security BearerAuth
func (h *EmployeeHandler) UpdateEmployee(c *gin.Context) {
	employeeIdStr := c.Param("employeeId")

	var payload models.UpdateEmployeePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	// 基本校验：确保至少提供了一个字段进行更新
	if payload.Department == nil && payload.EmploymentStatus == nil && payload.TerminationDate == nil {
		utils.RespondAPIError(c, http.StatusBadRequest, "至少需要提供一个更新字段", nil)
		return
	}

	// 业务逻辑校验：如果提供了 TerminationDate，EmploymentStatus 必须是 Departed
	// 或者如果 EmploymentStatus 更新为 Departed，TerminationDate 应该被处理（服务层会处理自动填充）
	if payload.TerminationDate != nil && *payload.TerminationDate != "" {
		if payload.EmploymentStatus == nil || (*payload.EmploymentStatus != "Departed" && *payload.EmploymentStatus != "") {
			// 如果提供了离职日期，但状态不是 Departed (也不是正在被更新为 Departed), 则这是一个无效组合
			// 除非业务允许仅更新离职日期而不改变状态（这比较少见）
			// 这里假设：如果提供了 terminationDate，则 employmentStatus 必须是 Departed，或者 payload 中 employmentStatus 也必须是 Departed。
			// 如果 employmentStatus 为空，但 terminationDate 有值，也视为不合法，因为不知道要不要把状态改为 Departed。
			// 更简单的处理是，如果 employmentStatus 不是 Departed，则不允许设置 terminationDate。
			// 若 employmentStatus 在 payload 中为 nil，则看数据库中当前员工状态是否为 Departed （这需要读一次员工信息，handler 层通常不做）
			// 为了简化 handler，主要的状态转换逻辑放在 service 层。Handler 层只做基本格式和必要组合校验。
			// 此处改为: 如果 TerminationDate 有值，则 EmploymentStatus 也必须有值且为 Departed。
			if payload.EmploymentStatus == nil || *payload.EmploymentStatus != "Departed" {
				utils.RespondAPIError(c, http.StatusBadRequest, "提供离职日期时，在职状态必须是 'Departed'", nil)
				return
			}
		}
	}
	// 如果 employmentStatus 更新为非 Departed，则 TerminationDate 不应该有值
	if payload.EmploymentStatus != nil && *payload.EmploymentStatus != "Departed" && payload.TerminationDate != nil && *payload.TerminationDate != "" {
		utils.RespondAPIError(c, http.StatusBadRequest, "在职状态不是 'Departed' 时，不应提供离职日期", nil)
		return
	}

	updatedEmployee, err := h.service.UpdateEmployee(employeeIdStr, payload) // 服务层接收 models.UpdateEmployeePayload
	if err != nil {
		if errors.Is(err, services.ErrEmployeeNotFound) {
			utils.RespondNotFoundError(c, "员工")
		} else if err.Error() == "没有提供任何有效的更新字段" || strings.Contains(err.Error(), "无效的离职日期格式") {
			utils.RespondAPIError(c, http.StatusBadRequest, err.Error(), nil)
		} else {
			utils.RespondInternalServerError(c, "更新员工信息失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusOK, updatedEmployee, "员工信息更新成功")
}

// BatchImportEmployees godoc
// @Summary 批量导入员工数据 (CSV)
// @Description 通过上传 CSV 文件批量导入员工。CSV文件应包含表头：fullName,phoneNumber,email,department。顺序必须一致，表头自身也会被计入行号。
// @Tags Employees
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "包含员工数据的 CSV 文件 (表头: fullName,phoneNumber,email,department)"
// @Success 200 {object} utils.SuccessResponse{data=BatchImportResponse} "导入结果摘要"
// @Failure 400 {object} utils.APIErrorResponse "请求错误，例如文件未提供、文件格式错误或CSV表头不匹配"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /employees/import [post]
// @Security BearerAuth
func (h *EmployeeHandler) BatchImportEmployees(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.RespondAPIError(c, http.StatusBadRequest, "无法读取上传的文件: "+err.Error(), nil)
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		utils.RespondAPIError(c, http.StatusBadRequest, "文件格式无效，请上传 CSV 文件", nil)
		return
	}

	reader := csv.NewReader(bufio.NewReader(file))
	var successCount, errorCount int
	var importErrors []BatchImportErrorDetail

	// 读取表头
	csvHeader, err := reader.Read()
	if err == io.EOF {
		utils.RespondAPIError(c, http.StatusBadRequest, "CSV 文件为空或只有表头", nil)
		return
	}
	if err != nil {
		utils.RespondAPIError(c, http.StatusBadRequest, "无法读取 CSV 表头: "+err.Error(), nil)
		return
	}

	// 校验表头 (fullName,phoneNumber,email,department)
	expectedHeader := []string{"fullName", "phoneNumber", "email", "department"}
	if !utils.CompareStringSlices(csvHeader, expectedHeader) {
		utils.RespondAPIError(c, http.StatusBadRequest, fmt.Sprintf("CSV 表头与预期不符。预期: %v, 得到: %v", expectedHeader, csvHeader), nil)
		return
	}

	rowNum := 1 // 从1开始计数，表头是第1行
	for {
		rowNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			importErrors = append(importErrors, BatchImportErrorDetail{RowNumber: rowNum, Reason: "无法读取行数据: " + err.Error()})
			errorCount++
			continue
		}

		if len(record) != len(expectedHeader) {
			importErrors = append(importErrors, BatchImportErrorDetail{RowNumber: rowNum, RowData: record, Reason: fmt.Sprintf("列数与表头不匹配，期望 %d 列，得到 %d 列", len(expectedHeader), len(record))})
			errorCount++
			continue
		}

		fullName := strings.TrimSpace(record[0])
		phoneNumberStr := strings.TrimSpace(record[1])
		emailStr := strings.TrimSpace(record[2])
		departmentStr := strings.TrimSpace(record[3])

		if fullName == "" {
			importErrors = append(importErrors, BatchImportErrorDetail{RowNumber: rowNum, RowData: record, Reason: "fullName 不能为空"})
			errorCount++
			continue
		}

		employeeToCreate := &models.Employee{
			FullName: fullName,
		}

		if phoneNumberStr != "" {
			employeeToCreate.PhoneNumber = &phoneNumberStr
		} else {
			employeeToCreate.PhoneNumber = nil // 明确设为nil，如果为空字符串
		}

		if emailStr != "" {
			// 可在此处添加 email 格式的基础校验，或依赖 service/model 层的校验
			employeeToCreate.Email = &emailStr
		} else {
			employeeToCreate.Email = nil
		}

		if departmentStr != "" {
			employeeToCreate.Department = &departmentStr
		} else {
			employeeToCreate.Department = nil
		}

		_, err = h.service.CreateEmployee(employeeToCreate) // service 层会自动生成 EmployeeID
		if err != nil {
			// 检查是否是已知的工号冲突错误 (由repo层转换而来)
			// 或者其他业务校验错误 (例如，如果未来手机或邮箱也要求唯一且冲突了)
			reason := err.Error()
			if errors.Is(err, repositories.ErrEmployeeIDExists) { // 理论上，由于ID是生成的，这个错误概率极低，但万一发生
				reason = "生成员工工号时发生冲突，请重试该行数据"
			} else if strings.Contains(err.Error(), "value too long for type character varying(50)") && strings.Contains(err.Error(), "phone_number") {
				reason = "phoneNumber 过长 (最大50字符)"
			} else if strings.Contains(err.Error(), "value too long for type character varying(255)") && strings.Contains(err.Error(), "email") {
				reason = "email 过长 (最大255字符)"
			} // 可以根据 service 层可能返回的错误类型添加更多处理

			importErrors = append(importErrors, BatchImportErrorDetail{RowNumber: rowNum, RowData: record, Reason: reason})
			errorCount++
		} else {
			successCount++
		}
	}

	response := BatchImportResponse{
		Message:      fmt.Sprintf("员工数据导入处理完成。成功: %d, 失败: %d", successCount, errorCount),
		SuccessCount: successCount,
		ErrorCount:   errorCount,
		Errors:       importErrors,
	}

	utils.RespondSuccess(c, http.StatusOK, response, response.Message)
}
