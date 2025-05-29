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
	"github.com/phone_management/internal/auth"
	"github.com/phone_management/internal/models"
	"github.com/phone_management/internal/repositories" // 用于判断 ErrPhoneNumberExists
	"github.com/phone_management/internal/services"
	"github.com/phone_management/pkg/utils"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// MobileNumberHandler 封装了手机号码相关的 HTTP 处理逻辑
type MobileNumberHandler struct {
	service services.MobileNumberService
}

// NewMobileNumberHandler 创建一个新的 MobileNumberHandler 实例
func NewMobileNumberHandler(service services.MobileNumberService) *MobileNumberHandler {
	return &MobileNumberHandler{service: service}
}

// CreateMobileNumberPayload 是用于绑定和验证创建手机号码请求的临时结构体
type CreateMobileNumberPayload struct {
	PhoneNumber         string  `json:"phoneNumber" binding:"required,max=50"`
	ApplicantEmployeeID string  `json:"applicantEmployeeId" binding:"required"` // 代表员工工号
	ApplicationDate     string  `json:"applicationDate" binding:"required,datetime=2006-01-02"`
	Status              string  `json:"status"`                                        // 移除状态验证，在业务层处理
	Purpose             *string `json:"purpose,omitempty" binding:"omitempty,max=255"` // 号码用途，可选
	Vendor              string  `json:"vendor" binding:"max=100"`
	Remarks             string  `json:"remarks" binding:"max=255"`
	// CurrentEmployeeDbID 和 CancellationDate 是可选的，如果它们在创建请求中也可能出现，也应在此处添加为字符串并处理
	// CurrentEmployeeDbID   *uint  `json:"currentEmployeeDbId,omitempty"`
	// CancellationDate      string `json:"cancellationDate,omitempty" binding:"omitempty,datetime=2006-01-02"`
}

// PagedMobileNumbersData 定义了手机号码列表的分页响应结构
type PagedMobileNumbersData struct {
	Items      []models.MobileNumberResponse `json:"items"`
	Pagination PaginationInfo                `json:"pagination"`
}

// PagedRiskNumbersData 定义了风险号码列表的分页响应结构
type PagedRiskNumbersData struct {
	Items      []models.RiskNumberResponse `json:"items"`
	Pagination PaginationInfo              `json:"pagination"`
}

// CreateMobileNumber godoc
// @Summary 新增一个手机号码
// @Description 从请求体绑定数据并验证，数据保存到 SQLite 的 MobileNumbers 表中，进行手机号码唯一性校验。
// @Tags MobileNumbers
// @Accept json
// @Produce json
// @Param mobileNumber body CreateMobileNumberPayload true "手机号码信息 (日期格式 YYYY-MM-DD)"
// @Success 201 {object} utils.SuccessResponse{data=models.MobileNumber} "创建成功的号码对象"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误或数据校验失败"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 409 {object} utils.APIErrorResponse "手机号码已存在"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /mobilenumbers [post]
// @Security BearerAuth
func (h *MobileNumberHandler) CreateMobileNumber(c *gin.Context) {
	var payload CreateMobileNumberPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	applicationDate, err := utils.ParseDate(payload.ApplicationDate)
	if err != nil {
		utils.RespondValidationError(c, "申请日期(applicationDate)格式无效: "+payload.ApplicationDate+", "+err.Error())
		return
	}

	// 如果没有提供状态，默认设置为闲置
	status := payload.Status
	if status == "" {
		status = string(models.StatusIdle)
	} else if !models.IsValidStatus(status) {
		utils.RespondAPIError(c, http.StatusBadRequest, "无效的状态值", "提供的状态: "+status)
		return
	}

	mobileNumberToCreate := &models.MobileNumber{
		PhoneNumber:         payload.PhoneNumber,
		ApplicantEmployeeID: payload.ApplicantEmployeeID,
		ApplicationDate:     applicationDate,
		Status:              status,
		Purpose:             payload.Purpose,
		Vendor:              payload.Vendor,
		Remarks:             payload.Remarks,
	}

	createdMobileNumber, err := h.service.CreateMobileNumber(mobileNumberToCreate)
	if err != nil {
		if errors.Is(err, repositories.ErrMobileNumberStringConflict) {
			utils.RespondConflictError(c, repositories.ErrMobileNumberStringConflict.Error())
		} else if errors.Is(err, services.ErrEmployeeNotFound) {
			utils.RespondAPIError(c, http.StatusNotFound, "办卡人员工工号未找到", "employeeId: "+payload.ApplicantEmployeeID)
		} else if errors.Is(err, utils.ErrInvalidPhoneNumberFormat) || errors.Is(err, utils.ErrInvalidPhoneNumberPrefix) {
			utils.RespondAPIError(c, http.StatusBadRequest, err.Error(), nil)
		} else {
			utils.RespondInternalServerError(c, "创建手机号码失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusCreated, createdMobileNumber, "手机号码创建成功")
}

// GetMobileNumbers godoc
// @Summary 获取手机号码列表
// @Description 根据查询参数获取手机号码列表，支持分页、搜索和筛选。注意：风险号码(risk_pending)通过专门的风险号码接口获取，此接口不返回风险号码。
// @Tags MobileNumbers
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Param sortBy query string false "排序字段 (例如: phoneNumber, applicationDate)"
// @Param sortOrder query string false "排序顺序 ('asc'或'desc')"
// @Param search query string false "搜索关键词 (匹配手机号、使用人、办卡人)"
// @Param status query string false "号码状态筛选 (例如: 闲置, 使用中，不包括风险号码)"
// @Param applicantStatus query string false "办卡人当前在职状态筛选 ('Active'或'Departed')"
// @Success 200 {object} utils.SuccessResponse{data=PagedMobileNumbersData} "成功响应，包含号码列表和分页信息"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /mobilenumbers [get]
// @Security BearerAuth
func (h *MobileNumberHandler) GetMobileNumbers(c *gin.Context) {
	type GetMobileNumbersQuery struct {
		Page            int    `form:"page,default=1"`
		Limit           int    `form:"limit,default=10"`
		SortBy          string `form:"sortBy"`
		SortOrder       string `form:"sortOrder,default=asc"`
		Search          string `form:"search"`
		Status          string `form:"status"`
		ApplicantStatus string `form:"applicantStatus"`
	}

	var queryParams GetMobileNumbersQuery
	if err := c.ShouldBindQuery(&queryParams); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	if queryParams.SortOrder != "asc" && queryParams.SortOrder != "desc" {
		queryParams.SortOrder = "asc"
	}
	if queryParams.Limit <= 0 {
		queryParams.Limit = 10
	}
	if queryParams.Page <= 0 {
		queryParams.Page = 1
	}

	mobileNumbers, totalItems, err := h.service.GetMobileNumbers(
		queryParams.Page,
		queryParams.Limit,
		queryParams.SortBy,
		queryParams.SortOrder,
		queryParams.Search,
		queryParams.Status,
		queryParams.ApplicantStatus,
	)

	if err != nil {
		utils.RespondInternalServerError(c, "获取手机号码列表失败", err.Error())
		return
	}

	totalPages := int64(0)
	if queryParams.Limit > 0 { // 防止除以零
		totalPages = (totalItems + int64(queryParams.Limit) - 1) / int64(queryParams.Limit)
	}
	if totalPages == 0 && totalItems > 0 {
		totalPages = 1
	}

	pagedData := PagedMobileNumbersData{
		Items: mobileNumbers,
		Pagination: PaginationInfo{
			TotalItems:  totalItems,
			TotalPages:  totalPages,
			CurrentPage: queryParams.Page,
			PageSize:    queryParams.Limit,
		},
	}

	utils.RespondSuccess(c, http.StatusOK, pagedData, "手机号码列表获取成功")
}

// GetMobileNumberByID godoc
// @Summary 获取指定手机号码的详情
// @Description 根据路径参数手机号码字符串获取单个手机号码的完整信息，包括其使用历史。
// @Tags MobileNumbers
// @Accept json
// @Produce json
// @Param phoneNumber path string true "手机号码字符串"
// @Success 200 {object} utils.SuccessResponse{data=models.MobileNumberResponse} "成功响应，包含号码详情及其使用历史"
// @Failure 400 {object} utils.APIErrorResponse "无效的手机号码格式 (保留，以防未来有格式校验)"
// @Failure 404 {object} utils.APIErrorResponse "号码未找到"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /mobilenumbers/{phoneNumber} [get]
// @Security BearerAuth
func (h *MobileNumberHandler) GetMobileNumberByID(c *gin.Context) {
	phoneNumberStr := c.Param("phoneNumber") // 读取 phoneNumber 字符串
	// 不再需要 parseUint
	// id, err := parseUint(idStr)
	// if err != nil {
	// 	utils.RespondAPIError(c, http.StatusBadRequest, "无效的ID格式", err.Error())
	// 	return
	// }

	// 假设服务层有 GetMobileNumberByPhoneNumberDetail 方法
	mobileNumber, err := h.service.GetMobileNumberByPhoneNumberDetail(phoneNumberStr)
	if err != nil {
		if errors.Is(err, services.ErrMobileNumberNotFound) {
			utils.RespondNotFoundError(c, "手机号码")
		} else {
			utils.RespondInternalServerError(c, "获取手机号码详情失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusOK, mobileNumber, "手机号码详情获取成功")
}

// UpdateMobileNumber godoc
// @Summary 更新指定手机号码的信息
// @Description 更新指定手机号码的信息 (主要用于更新状态、用途、供应商、备注)。当号码状态变更为"已注销"时，自动记录注销时间。注意：风险号码(risk_pending)不允许通过此接口更新，请使用专门的风险处理接口。
// @Tags MobileNumbers
// @Accept json
// @Produce json
// @Param phoneNumber path string true "手机号码字符串"
// @Param mobileNumberUpdate body models.MobileNumberUpdatePayload true "要更新的手机号码字段"
// @Success 200 {object} utils.SuccessResponse{data=models.MobileNumber} "更新后的号码对象"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误或数据校验失败 / 没有提供任何更新字段 / 无效的手机号码格式"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 403 {object} utils.APIErrorResponse "风险号码不允许通过常规接口更新"
// @Failure 404 {object} utils.APIErrorResponse "号码未找到"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /mobilenumbers/{phoneNumber}/update [post]
// @Security BearerAuth
func (h *MobileNumberHandler) UpdateMobileNumber(c *gin.Context) {
	phoneNumberStr := c.Param("phoneNumber") // 读取 phoneNumber 字符串

	// 首先验证手机号码格式
	if err := utils.ValidatePhoneNumber(phoneNumberStr); err != nil {
		utils.RespondAPIError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	var payload models.MobileNumberUpdatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	// 校验至少有一个字段被提供用于更新
	if payload.Status == nil && payload.Purpose == nil && payload.Vendor == nil && payload.Remarks == nil {
		utils.RespondAPIError(c, http.StatusBadRequest, "没有提供任何有效的更新字段", nil)
		return
	}

	// 假设服务层有 UpdateMobileNumberByPhoneNumber 方法
	updatedMobileNumber, err := h.service.UpdateMobileNumberByPhoneNumber(phoneNumberStr, payload)
	if err != nil {
		if errors.Is(err, services.ErrMobileNumberNotFound) {
			utils.RespondNotFoundError(c, "手机号码")
		} else if err.Error() == "没有提供任何更新字段" {
			utils.RespondAPIError(c, http.StatusBadRequest, err.Error(), nil)
		} else if err.Error() == "无效的状态值" {
			utils.RespondAPIError(c, http.StatusBadRequest, err.Error(), nil)
		} else if strings.Contains(err.Error(), "不能直接将状态更新为'使用中'") {
			utils.RespondAPIError(c, http.StatusBadRequest, err.Error(), "请使用分配接口 /assign 来将号码分配给用户")
		} else if strings.Contains(err.Error(), "号码当前为'使用中'状态") {
			utils.RespondAPIError(c, http.StatusConflict, err.Error(), "请使用回收接口 /unassign 来回收号码后再进行状态更新")
		} else if strings.Contains(err.Error(), "风险号码不允许通过常规更新接口修改") {
			utils.RespondAPIError(c, http.StatusForbidden, "风险号码不允许通过常规更新接口修改", "请使用专门的风险处理接口 /handle-risk")
		} else {
			utils.RespondInternalServerError(c, "更新手机号码失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusOK, updatedMobileNumber, "手机号码更新成功")
}

// AssignMobileNumber godoc
// @Summary 将指定手机号码分配给一个员工
// @Description 校验目标号码是否为"闲置"状态，目标员工是否为"在职"状态。更新号码记录，关联当前使用人员工ID，将号码状态改为"使用中"。创建一条新的号码使用历史记录。
// @Tags MobileNumbers
// @Accept json
// @Produce json
// @Param phoneNumber path string true "手机号码字符串"
// @Param assignPayload body models.MobileNumberAssignPayload true "分配信息 (员工业务工号和分配日期和用途 YYYY-MM-DD)"
// @Success 200 {object} utils.SuccessResponse{data=models.MobileNumber} "成功分配后的号码对象"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误 / 无效的日期格式 / 无效的手机号码格式" // 更新了描述
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 404 {object} utils.APIErrorResponse "手机号码或目标员工工号未找到"
// @Failure 409 {object} utils.APIErrorResponse "操作冲突 (例如：号码非闲置，员工非在职)"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /mobilenumbers/{phoneNumber}/assign [post]
// @Security BearerAuth
func (h *MobileNumberHandler) AssignMobileNumber(c *gin.Context) {
	phoneNumberStr := c.Param("phoneNumber") // 从 :phoneNumber 获取

	// 首先验证手机号码格式
	if err := utils.ValidatePhoneNumber(phoneNumberStr); err != nil {
		utils.RespondAPIError(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	var payload models.MobileNumberAssignPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	assignmentDate, err := utils.ParseDate(payload.AssignmentDate)
	if err != nil {
		utils.RespondAPIError(c, http.StatusBadRequest, "分配日期(assignmentDate)格式无效: "+err.Error(), nil)
		return
	}

	// 调用服务层，传递 phoneNumberStr 而不是 numberID，并添加用途字段
	assignedMobileNumber, err := h.service.AssignMobileNumber(phoneNumberStr, payload.EmployeeID, assignmentDate, payload.Purpose)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMobileNumberNotFound):
			utils.RespondNotFoundError(c, "手机号码")
		case errors.Is(err, services.ErrEmployeeNotFound): // 来自于 EmployeeService.GetEmployeeByBusinessID
			utils.RespondAPIError(c, http.StatusNotFound, "目标员工未找到 (基于提供的工号)", "employeeId: "+payload.EmployeeID)
		case errors.Is(err, repositories.ErrMobileNumberNotInIdleStatus):
			utils.RespondAPIError(c, http.StatusConflict, "手机号码不是闲置状态，无法分配", err.Error())
		case errors.Is(err, repositories.ErrEmployeeNotActive):
			utils.RespondAPIError(c, http.StatusConflict, "目标员工不是在职状态，无法分配", err.Error())
		default:
			utils.RespondInternalServerError(c, "分配手机号码失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusOK, assignedMobileNumber, "手机号码分配成功")
}

// UnassignMobileNumber godoc
// @Summary 从当前使用人处回收指定手机号码
// @Description 回收手机号码，支持在用(in_use)、风险待核实(risk_pending)、用户报告(user_reported)状态的号码。对于有当前使用人的号码，会更新使用历史记录的结束时间；对于没有当前使用人的号码，直接设为闲置状态。
// @Tags MobileNumbers
// @Accept json
// @Produce json
// @Param phoneNumber path string true "手机号码字符串"
// @Param unassignPayload body models.MobileNumberUnassignPayload false "回收信息 (可选，包含回收日期 YYYY-MM-DD)"
// @Success 200 {object} utils.SuccessResponse{data=models.MobileNumber} "成功回收后的号码对象"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误 / 无效的日期格式 / 无效的手机号码格式"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 404 {object} utils.APIErrorResponse "手机号码未找到"
// @Failure 409 {object} utils.APIErrorResponse "操作冲突 (例如：号码状态不允许回收，或未找到有效的分配记录)"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /mobilenumbers/{phoneNumber}/unassign [post]
// @Security BearerAuth
func (h *MobileNumberHandler) UnassignMobileNumber(c *gin.Context) {
	phoneNumberStr := c.Param("phoneNumber") // 读取 phoneNumber 字符串

	// 首先验证手机号码格式
	if err := utils.ValidatePhoneNumber(phoneNumberStr); err != nil {
		// utils.RespondValidationError(c, err.Error()) // 这个也可以，但 RespondAPIError 更通用
		utils.RespondAPIError(c, http.StatusBadRequest, err.Error(), nil) // 使用 validator 中定义的错误信息
		return
	}

	var payload models.MobileNumberUnassignPayload
	// 注意：c.ShouldBindJSON 即使在没有请求体时也不会报错，除非明确要求非空请求体 (e.g. `binding:"required"`)
	// 对于可选的请求体，如果解析失败（例如 JSON 格式错误），它会报错。
	// 如果请求体为空，payload 会是其零值，这对于可选 payload 是正常的。
	if err := c.ShouldBindJSON(&payload); err != nil {
		// 只有在 JSON 格式确实有问题时才报错，而不是因为它是空的。
		// 如果允许空 body，但 body 中有非法的 JSON，这里会捕获。
		// 对于完全没有 body 的情况，ShouldBindJSON 不会报错（除非有 `binding:"required"`）。
		// 如果希望对空 body 或格式错误的 body 都进行严格校验，可以在此添加逻辑。
		// 但通常，对于可选 body，我们只关心它是否提供了以及是否格式正确。
		utils.RespondValidationError(c, err.Error())
		return
	}

	reclaimDate := time.Now() // 默认为当前时间
	if payload.ReclaimDate != "" {
		parsedDate, err := utils.ParseDate(payload.ReclaimDate)
		if err != nil {
			utils.RespondAPIError(c, http.StatusBadRequest, "回收日期(reclaimDate)格式无效: "+err.Error(), nil)
			return
		}
		reclaimDate = parsedDate
	}

	// 假设服务层有 UnassignMobileNumberByPhoneNumber 方法
	unassignedMobileNumber, err := h.service.UnassignMobileNumberByPhoneNumber(phoneNumberStr, reclaimDate)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMobileNumberNotFound):
			utils.RespondNotFoundError(c, "手机号码")
		case errors.Is(err, repositories.ErrMobileNumberNotRecoverable):
			utils.RespondAPIError(c, http.StatusConflict, "手机号码状态不允许回收", "仅支持在用、风险待核实、用户报告状态的号码回收")
		case errors.Is(err, repositories.ErrNoActiveUsageHistoryFound):
			utils.RespondAPIError(c, http.StatusConflict, "未找到该号码当前有效的分配记录", err.Error())
		case strings.Contains(err.Error(), "数据不一致：使用号码没有关联当前用户"): // 这个错误可能来自 service 层更深处
			utils.RespondAPIError(c, http.StatusInternalServerError, "服务器内部错误: 数据不一致", err.Error())
		default:
			utils.RespondInternalServerError(c, "回收手机号码失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusOK, unassignedMobileNumber, "手机号码回收成功")
}

// BatchImportMobileNumberErrorDetail 描述了批量导入手机号码中单行数据的错误信息
// (与员工导入的 BatchImportErrorDetail 结构相同，可以考虑提取到公共 utils 或 handlers/common_types.go)
type BatchImportMobileNumberErrorDetail struct {
	RowNumber int      `json:"rowNumber"`         // CSV中的原始行号 (从1开始计数，包括表头)
	RowData   []string `json:"rowData,omitempty"` // 可选，原始行数据
	Reason    string   `json:"reason"`            // 错误原因
}

// BatchImportMobileNumbersResponse 定义了批量导入手机号码的响应结构
type BatchImportMobileNumbersResponse struct {
	Message      string                               `json:"message"`
	SuccessCount int                                  `json:"successCount"`
	ErrorCount   int                                  `json:"errorCount"`
	Errors       []BatchImportMobileNumberErrorDetail `json:"errors,omitempty"`
}

// BatchImportMobileNumbers godoc
// @Summary 批量导入手机号码数据 (CSV)
// @Description 通过上传 CSV 文件批量导入手机号码。CSV文件应包含表头：phoneNumber,applicantName,applicationDate,vendor
// @Tags MobileNumbers
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "包含手机号码数据的 CSV 文件 (表头: phoneNumber,applicantName,applicationDate,vendor)"
// @Success 200 {object} utils.SuccessResponse{data=BatchImportMobileNumbersResponse} "导入结果摘要"
// @Failure 400 {object} utils.APIErrorResponse "请求错误，例如文件未提供、文件格式错误或CSV表头不匹配"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /mobilenumbers/import [post]
// @Security BearerAuth
func (h *MobileNumberHandler) BatchImportMobileNumbers(c *gin.Context) {
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
	var importErrors []BatchImportMobileNumberErrorDetail

	csvHeader, err := reader.Read()
	if err == io.EOF {
		utils.RespondAPIError(c, http.StatusBadRequest, "CSV 文件为空或只有表头", nil)
		return
	}
	if err != nil {
		utils.RespondAPIError(c, http.StatusBadRequest, "无法读取 CSV 表头: "+err.Error(), nil)
		return
	}

	// 兼容 UTF-8 BOM
	if len(csvHeader) > 0 {
		csvHeader[0] = strings.TrimPrefix(csvHeader[0], "\uFEFF")
	}

	expectedHeader := []string{"phoneNumber", "applicantName", "applicationDate", "vendor"}
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
			importErrors = append(importErrors, BatchImportMobileNumberErrorDetail{RowNumber: rowNum, Reason: "无法读取行数据: " + err.Error()})
			errorCount++
			continue
		}

		if len(record) != len(expectedHeader) {
			importErrors = append(importErrors, BatchImportMobileNumberErrorDetail{RowNumber: rowNum, RowData: record, Reason: fmt.Sprintf("列数与表头不匹配，期望 %d 列，得到 %d 列", len(expectedHeader), len(record))})
			errorCount++
			continue
		}

		phoneNumberStr := strings.TrimSpace(record[0])
		applicantName := strings.TrimSpace(record[1])
		applicationDateStr := strings.TrimSpace(record[2])
		vendorStr := strings.TrimSpace(record[3])

		if phoneNumberStr == "" {
			importErrors = append(importErrors, BatchImportMobileNumberErrorDetail{RowNumber: rowNum, RowData: record, Reason: "phoneNumber 不能为空"})
			errorCount++
			continue
		} else if err := utils.ValidatePhoneNumber(phoneNumberStr); err != nil {
			importErrors = append(importErrors, BatchImportMobileNumberErrorDetail{RowNumber: rowNum, RowData: record, Reason: err.Error()})
			errorCount++
			continue
		}

		if applicantName == "" {
			importErrors = append(importErrors, BatchImportMobileNumberErrorDetail{RowNumber: rowNum, RowData: record, Reason: "applicantName 不能为空"})
			errorCount++
			continue
		}

		var applicationDate time.Time
		if applicationDateStr == "" {
			importErrors = append(importErrors, BatchImportMobileNumberErrorDetail{RowNumber: rowNum, RowData: record, Reason: "applicationDate 不能为空"})
			errorCount++
			continue
		} else {
			parsedDate, errDate := utils.ParseDate(applicationDateStr)
			if errDate != nil {
				importErrors = append(importErrors, BatchImportMobileNumberErrorDetail{RowNumber: rowNum, RowData: record, Reason: errDate.Error()})
				errorCount++
				continue
			}
			applicationDate = parsedDate
		}

		applicantEmployeeID, resolveErr := h.service.ResolveApplicantNameToID(applicantName)
		if resolveErr != nil {
			importErrors = append(importErrors, BatchImportMobileNumberErrorDetail{RowNumber: rowNum, RowData: record, Reason: resolveErr.Error()})
			errorCount++
			continue
		}

		mobileToCreate := &models.MobileNumber{
			PhoneNumber:         phoneNumberStr,
			ApplicantEmployeeID: applicantEmployeeID,
			ApplicationDate:     applicationDate,
			Status:              string(models.StatusIdle),
			Vendor:              vendorStr,
			Remarks:             "",
		}

		_, createErr := h.service.CreateMobileNumber(mobileToCreate)
		if createErr != nil {
			importErrors = append(importErrors, BatchImportMobileNumberErrorDetail{RowNumber: rowNum, RowData: record, Reason: createErr.Error()})
			errorCount++
		} else {
			successCount++
		}
	}

	response := BatchImportMobileNumbersResponse{
		Message:      fmt.Sprintf("手机号码数据导入处理完成。成功: %d, 失败: %d", successCount, errorCount),
		SuccessCount: successCount,
		ErrorCount:   errorCount,
		Errors:       importErrors,
	}

	utils.RespondSuccess(c, http.StatusOK, response, response.Message)
}

// GetRiskPendingNumbers godoc
// @Summary 获取风险号码列表
// @Description 获取状态为risk_pending的手机号码列表，支持分页、搜索和筛选
// @Tags MobileNumbers
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Param sortBy query string false "排序字段 (例如: phoneNumber, applicationDate, status)"
// @Param sortOrder query string false "排序顺序 ('asc'或'desc')" default("desc")
// @Param search query string false "搜索关键词 (匹配手机号、办卡人姓名、当前使用人姓名)"
// @Param applicantStatus query string false "办卡人在职状态筛选 ('Active'或'Departed')"
// @Success 200 {object} utils.SuccessResponse{data=PagedRiskNumbersData} "成功响应，包含风险号码列表和分页信息"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /mobilenumbers/risk-pending [get]
// @Security BearerAuth
func (h *MobileNumberHandler) GetRiskPendingNumbers(c *gin.Context) {
	type GetRiskPendingNumbersQuery struct {
		Page            int    `form:"page,default=1"`
		Limit           int    `form:"limit,default=10"`
		SortBy          string `form:"sortBy"`
		SortOrder       string `form:"sortOrder,default=desc"`
		Search          string `form:"search"`
		ApplicantStatus string `form:"applicantStatus" binding:"omitempty,oneof=Active Departed"`
	}

	var queryParams GetRiskPendingNumbersQuery
	if err := c.ShouldBindQuery(&queryParams); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	if queryParams.SortOrder != "asc" && queryParams.SortOrder != "desc" {
		queryParams.SortOrder = "desc"
	}
	if queryParams.Limit <= 0 {
		queryParams.Limit = 10
	}
	if queryParams.Page <= 0 {
		queryParams.Page = 1
	}

	riskNumbers, totalItems, err := h.service.GetRiskPendingNumbers(
		queryParams.Page,
		queryParams.Limit,
		queryParams.SortBy,
		queryParams.SortOrder,
		queryParams.Search,
		queryParams.ApplicantStatus,
	)

	if err != nil {
		utils.RespondInternalServerError(c, "获取风险号码列表失败", err.Error())
		return
	}

	totalPages := int64(0)
	if queryParams.Limit > 0 {
		totalPages = (totalItems + int64(queryParams.Limit) - 1) / int64(queryParams.Limit)
	}
	if totalPages == 0 && totalItems > 0 {
		totalPages = 1
	}

	pagedData := PagedRiskNumbersData{
		Items: riskNumbers,
		Pagination: PaginationInfo{
			TotalItems:  totalItems,
			TotalPages:  totalPages,
			CurrentPage: queryParams.Page,
			PageSize:    queryParams.Limit,
		},
	}

	utils.RespondSuccess(c, http.StatusOK, pagedData, "风险号码列表获取成功")
}

// HandleRiskNumber godoc
// @Summary 处理风险号码
// @Description 处理状态为risk_pending的号码，支持变更办卡人、回收号码、注销号码三种操作
// @Tags MobileNumbers
// @Accept json
// @Produce json
// @Param phoneNumber path string true "手机号码字符串"
// @Param handleRisk body models.HandleRiskNumberPayload true "处理风险号码的请求体"
// @Success 200 {object} utils.SuccessResponse{data=models.MobileNumber} "处理成功的号码对象"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误、数据校验失败或业务逻辑错误"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 404 {object} utils.APIErrorResponse "手机号码未找到或员工未找到"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /mobilenumbers/{phoneNumber}/handle-risk [post]
// @Security BearerAuth
func (h *MobileNumberHandler) HandleRiskNumber(c *gin.Context) {
	phoneNumberStr := c.Param("phoneNumber")

	var payload models.HandleRiskNumberPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	// 验证Action类型是否有效
	if !models.IsValidRiskHandleAction(payload.Action) {
		utils.RespondAPIError(c, http.StatusBadRequest, "无效的操作类型", "支持的操作: change_applicant, reclaim, deactivate")
		return
	}

	// 业务逻辑校验：变更办卡人时必须提供新办卡人员工ID
	if payload.Action == string(models.ActionChangeApplicant) {
		if payload.NewApplicantEmployeeID == nil || *payload.NewApplicantEmployeeID == "" {
			utils.RespondAPIError(c, http.StatusBadRequest, "变更办卡人时必须提供新办卡人员工ID", nil)
			return
		}
	}

	// 从JWT token中获取当前操作员用户名
	operatorUsername, ok := auth.GetCurrentUsername(c)
	if !ok {
		utils.RespondInternalServerError(c, "无法获取当前操作员信息", "用户上下文信息缺失")
		return
	}

	handledNumber, err := h.service.HandleRiskNumber(phoneNumberStr, payload, operatorUsername)
	if err != nil {
		if errors.Is(err, services.ErrMobileNumberNotFound) {
			utils.RespondNotFoundError(c, "手机号码")
		} else if errors.Is(err, services.ErrEmployeeNotFound) {
			utils.RespondAPIError(c, http.StatusBadRequest, "新办卡人员工未找到", nil)
		} else if errors.Is(err, repositories.ErrEmployeeNotActive) {
			utils.RespondAPIError(c, http.StatusBadRequest, "新办卡人员工不是在职状态", nil)
		} else if strings.Contains(err.Error(), "只能处理状态为 risk_pending 的号码") {
			utils.RespondAPIError(c, http.StatusBadRequest, "只能处理状态为风险待核实(risk_pending)的号码", nil)
		} else if strings.Contains(err.Error(), "变更办卡人时必须提供新办卡人员工ID") ||
			strings.Contains(err.Error(), "新办卡人员工未找到") ||
			strings.Contains(err.Error(), "新办卡人必须是在职状态") ||
			strings.Contains(err.Error(), "无效的操作类型") {
			utils.RespondAPIError(c, http.StatusBadRequest, err.Error(), nil)
		} else {
			utils.RespondInternalServerError(c, "处理风险号码失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusOK, handledNumber, "风险号码处理成功")
}
