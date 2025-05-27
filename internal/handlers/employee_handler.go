package handlers

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phone_management/internal/models"
	"github.com/phone_management/internal/repositories"
	"github.com/phone_management/internal/services"
	"github.com/phone_management/pkg/utils"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// EmployeeHandler 封装了员工相关的 HTTP 处理逻辑
type EmployeeHandler struct {
	service services.EmployeeService
}

// NewEmployeeHandler 创建一个新的 EmployeeHandler 实例
func NewEmployeeHandler(service services.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{service: service}
}

// CreateEmployeePayload 定义了创建员工请求的 JSON 结构体
// 注意：EmployeeID 由系统自动生成，EmploymentStatus 默认为 "Active"
type CreateEmployeePayload struct {
	FullName    string  `json:"fullName" binding:"required,max=255"`
	PhoneNumber *string `json:"phoneNumber,omitempty" binding:"omitempty,len=11,numeric"` // 手机号：可选，但如果提供，必须是11位数字
	Email       *string `json:"email,omitempty" binding:"omitempty,email,max=255"`        // 可选，需要是合法的email格式，最大长度255
	Department  *string `json:"department,omitempty" binding:"omitempty,max=255"`
	HireDate    *string `json:"hireDate,omitempty" binding:"omitempty,datetime=2006-01-02"` // 入职日期，可选，格式 YYYY-MM-DD
	// EmploymentStatus 默认为 "Active"，在模型或服务层处理，此处不需传递
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
// @Description 从请求体绑定数据并验证，数据保存到数据库。员工工号由系统自动生成。支持设置可选的入职日期（格式：YYYY-MM-DD）。
// @Tags Employees
// @Accept json
// @Produce json
// @Param employee body CreateEmployeePayload true "员工信息。包含必填的姓名，可选的手机号、邮箱、部门和入职日期"
// @Success 201 {object} utils.SuccessResponse{data=models.Employee} "创建成功的员工对象"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误、数据校验失败或入职日期格式无效"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 409 {object} utils.APIErrorResponse "手机号或邮箱已存在"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /employees [post]
// @Security BearerAuth
func (h *EmployeeHandler) CreateEmployee(c *gin.Context) {
	var payload CreateEmployeePayload
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

	// 处理入职日期
	if payload.HireDate != nil && *payload.HireDate != "" {
		hireDate, err := time.Parse("2006-01-02", *payload.HireDate)
		if err != nil {
			utils.RespondAPIError(c, http.StatusBadRequest, "无效的入职日期格式: "+*payload.HireDate, nil)
			return
		}
		employeeToCreate.HireDate = &hireDate
	}

	createdEmployee, err := h.service.CreateEmployee(employeeToCreate)
	if err != nil {
		// 处理来自服务层的唯一性冲突错误
		if errors.Is(err, services.ErrPhoneNumberExists) || errors.Is(err, services.ErrEmailExists) || errors.Is(err, repositories.ErrEmployeeIDExists) {
			utils.RespondConflictError(c, err.Error())
			// 处理来自服务层（通过 utils 包传递）的格式错误
		} else if errors.Is(err, utils.ErrInvalidPhoneNumberFormat) || errors.Is(err, utils.ErrInvalidPhoneNumberPrefix) {
			utils.RespondAPIError(c, http.StatusBadRequest, err.Error(), nil)
			// 可选：如果 service 层也可能返回 utils.ErrInvalidEmailFormat (目前仅在handler的批量导入中校验)
			// } else if errors.Is(err, utils.ErrInvalidEmailFormat) {
			// 	 utils.RespondAPIError(c, http.StatusBadRequest, err.Error(), nil)
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
// @Description 根据员工业务工号更新员工的部门、入职日期、在职状态或离职日期。所有字段都是可选的，至少需要提供一个字段进行更新。入职日期和离职日期格式为 YYYY-MM-DD。在职状态允许值为 'Active' 或 'Departed'。
// @Tags Employees
// @Accept json
// @Produce json
// @Param employeeId path string true "员工业务工号"
// @Param employeeUpdate body models.UpdateEmployeePayload true "要更新的员工字段。可包含部门、入职日期、在职状态、离职日期"
// @Success 200 {object} utils.SuccessResponse{data=models.Employee} "更新后的员工对象"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误、数据校验失败、日期格式无效或业务逻辑错误"
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
	if payload.Department == nil && payload.EmploymentStatus == nil && payload.HireDate == nil && payload.TerminationDate == nil {
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
// @Description 通过上传 CSV 文件批量导入员工。CSV文件必须包含表头：fullName,phoneNumber,email,department,hireDate。列顺序必须一致。fullName为必填，其他字段可为空。hireDate格式为YYYY-MM-DD。支持GBK和UTF-8编码。
// @Tags Employees
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "包含员工数据的 CSV 文件。表头: fullName,phoneNumber,email,department,hireDate"
// @Success 200 {object} utils.SuccessResponse{data=BatchImportResponse} "导入结果摘要，包含成功和失败的详细信息"
// @Failure 400 {object} utils.APIErrorResponse "请求错误，例如文件未提供、文件格式错误、CSV表头不匹配或数据格式错误"
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

	// 兼容GBK和UTF-8编码
	reader := csv.NewReader(transform.NewReader(file, simplifiedchinese.GBK.NewDecoder()))
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
	// 兼容 UTF-8 BOM (如果需要，已在 mobilenumber_handler 中添加，此处可以考虑也加上)
	if len(csvHeader) > 0 {
		csvHeader[0] = strings.TrimPrefix(csvHeader[0], "\uFEFF")
	}

	expectedHeader := []string{"fullName", "phoneNumber", "email", "department", "hireDate"}
	if !utils.CompareStringSlices(csvHeader, expectedHeader) {
		utils.RespondAPIError(c, http.StatusBadRequest, fmt.Sprintf("CSV 表头与预期不符。预期: %v, 得到: %v", expectedHeader, csvHeader), nil)
		return
	}

	rowNum := 1
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
		hireDateStr := strings.TrimSpace(record[4])

		if fullName == "" {
			importErrors = append(importErrors, BatchImportErrorDetail{RowNumber: rowNum, RowData: record, Reason: "fullName 不能为空"})
			errorCount++
			continue
		}

		employeeToCreate := &models.Employee{
			FullName: fullName,
		}

		if phoneNumberStr != "" {
			if err := utils.ValidatePhoneNumber(phoneNumberStr); err != nil { // 使用 utils.ValidatePhoneNumber
				importErrors = append(importErrors, BatchImportErrorDetail{RowNumber: rowNum, RowData: record, Reason: err.Error()})
				errorCount++
				continue
			}
			employeeToCreate.PhoneNumber = &phoneNumberStr
		} else {
			employeeToCreate.PhoneNumber = nil
		}

		if emailStr != "" {
			if !utils.ValidateEmailFormat(emailStr) { // 使用 utils.ValidateEmailFormat
				importErrors = append(importErrors, BatchImportErrorDetail{RowNumber: rowNum, RowData: record, Reason: utils.ErrInvalidEmailFormat.Error()})
				errorCount++
				continue
			}
			employeeToCreate.Email = &emailStr
		} else {
			employeeToCreate.Email = nil
		}

		if departmentStr != "" {
			employeeToCreate.Department = &departmentStr
		} else {
			employeeToCreate.Department = nil
		}

		if hireDateStr != "" {
			hireDate, err := time.Parse("2006-01-02", hireDateStr)
			if err != nil {
				importErrors = append(importErrors, BatchImportErrorDetail{RowNumber: rowNum, RowData: record, Reason: "无效的入职日期格式: " + hireDateStr})
				errorCount++
				continue
			}
			employeeToCreate.HireDate = &hireDate
		} else {
			employeeToCreate.HireDate = nil
		}

		_, err = h.service.CreateEmployee(employeeToCreate)
		if err != nil {
			reason := err.Error()
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
