package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phone_management/internal/models"
	"github.com/phone_management/internal/repositories" // 用于判断 ErrPhoneNumberExists
	"github.com/phone_management/internal/services"
	"github.com/phone_management/pkg/utils" // 新增导入
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
	PhoneNumber         string `json:"phoneNumber" binding:"required,max=50"`
	ApplicantEmployeeID string `json:"applicantEmployeeId" binding:"required"` // 改为 string，代表业务工号
	ApplicationDate     string `json:"applicationDate" binding:"required,datetime=2006-01-02"`
	Status              string `json:"status" binding:"required,oneof=闲置 在用 待注销 已注销 待核实-办卡人离职"`
	Vendor              string `json:"vendor" binding:"max=100"`
	Remarks             string `json:"remarks" binding:"max=255"`
	// CurrentEmployeeDbID 和 CancellationDate 是可选的，如果它们在创建请求中也可能出现，也应在此处添加为字符串并处理
	// CurrentEmployeeDbID   *uint  `json:"currentEmployeeDbId,omitempty"`
	// CancellationDate      string `json:"cancellationDate,omitempty" binding:"omitempty,datetime=2006-01-02"`
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

	applicationDate, err := time.Parse("2006-01-02", payload.ApplicationDate)
	if err != nil {
		utils.RespondValidationError(c, "申请日期格式无效: "+err.Error())
		return
	}

	mobileNumberToCreate := &models.MobileNumber{
		PhoneNumber:         payload.PhoneNumber,
		ApplicantEmployeeID: payload.ApplicantEmployeeID, // 直接使用业务工号
		ApplicationDate:     applicationDate,
		Status:              payload.Status,
		Vendor:              payload.Vendor,
		Remarks:             payload.Remarks,
	}

	// 服务层 CreateMobileNumber 方法签名已更新，不再需要第二个 applicantBusinessID 参数
	createdMobileNumber, err := h.service.CreateMobileNumber(mobileNumberToCreate)
	if err != nil {
		if errors.Is(err, repositories.ErrMobileNumberStringConflict) {
			utils.RespondConflictError(c, repositories.ErrMobileNumberStringConflict.Error())
		} else if errors.Is(err, services.ErrEmployeeNotFound) {
			utils.RespondAPIError(c, http.StatusNotFound, "办卡人员工工号未找到", "employeeId: "+payload.ApplicantEmployeeID)
		} else {
			utils.RespondInternalServerError(c, "创建手机号码失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusCreated, createdMobileNumber, "手机号码创建成功")
}

// 定义 GetMobileNumbers 的分页响应结构
type PagedMobileNumbersData struct {
	Items      []models.MobileNumberResponse `json:"items"`
	Pagination PaginationInfo                `json:"pagination"`
}

// GetMobileNumbers godoc
// @Summary 获取手机号码列表
// @Description 根据查询参数获取手机号码列表，支持分页、搜索和筛选
// @Tags MobileNumbers
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Param sortBy query string false "排序字段 (例如: phoneNumber, applicationDate)"
// @Param sortOrder query string false "排序顺序 ('asc'或'desc')"
// @Param search query string false "搜索关键词 (匹配手机号、使用人、办卡人)"
// @Param status query string false "号码状态筛选 (例如: 闲置, 在用)"
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
// @Description 更新指定手机号码的信息 (主要用于更新状态、供应商、备注)。当号码状态变更为"已注销"时，自动记录注销时间。
// @Tags MobileNumbers
// @Accept json
// @Produce json
// @Param phoneNumber path string true "手机号码字符串"
// @Param mobileNumberUpdate body models.MobileNumberUpdatePayload true "要更新的手机号码字段"
// @Success 200 {object} utils.SuccessResponse{data=models.MobileNumber} "更新后的号码对象"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误或数据校验失败 / 没有提供任何更新字段"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 404 {object} utils.APIErrorResponse "号码未找到"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /mobilenumbers/{phoneNumber}/update [post]
// @Security BearerAuth
func (h *MobileNumberHandler) UpdateMobileNumber(c *gin.Context) {
	phoneNumberStr := c.Param("phoneNumber") // 读取 phoneNumber 字符串
	//不再需要 parseUint
	// id, err := parseUint(idStr)
	// if err != nil {
	// 	utils.RespondAPIError(c, http.StatusBadRequest, "无效的ID格式", err.Error())
	// 	return
	// }

	var payload models.MobileNumberUpdatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	if payload.Status == nil && payload.Vendor == nil && payload.Remarks == nil {
		utils.RespondAPIError(c, http.StatusBadRequest, "没有提供任何有效的更新字段", nil)
		return
	}

	// 假设服务层有 UpdateMobileNumberByPhoneNumber 方法
	updatedMobileNumber, err := h.service.UpdateMobileNumberByPhoneNumber(phoneNumberStr, payload)
	if err != nil {
		if errors.Is(err, services.ErrMobileNumberNotFound) {
			utils.RespondNotFoundError(c, "手机号码")
		} else if err.Error() == "没有提供任何更新字段" { // 这个错误来自 service 层
			utils.RespondAPIError(c, http.StatusBadRequest, err.Error(), nil)
		} else {
			utils.RespondInternalServerError(c, "更新手机号码失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusOK, updatedMobileNumber, "手机号码更新成功")
}

// MobileNumberAssignPayload 定义了分配号码的请求体
type MobileNumberAssignPayload struct {
	EmployeeID     string `json:"employeeId" binding:"required"` // 改为 string, 代表业务工号
	AssignmentDate string `json:"assignmentDate" binding:"required,datetime=2006-01-02"`
}

// AssignMobileNumber godoc
// @Summary 将指定手机号码分配给一个员工
// @Description 校验目标号码是否为"闲置"状态，目标员工是否为"在职"状态。更新号码记录，关联当前使用人员工ID，将号码状态改为"在用"。创建一条新的号码使用历史记录。
// @Tags MobileNumbers
// @Accept json
// @Produce json
// @Param phoneNumber path string true "手机号码字符串"
// @Param assignPayload body models.MobileNumberAssignPayload true "分配信息 (员工业务工号和分配日期 YYYY-MM-DD)"
// @Success 200 {object} utils.SuccessResponse{data=models.MobileNumber} "成功分配后的号码对象"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误 / 无效的日期格式"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 404 {object} utils.APIErrorResponse "手机号码或目标员工工号未找到"
// @Failure 409 {object} utils.APIErrorResponse "操作冲突 (例如：号码非闲置，员工非在职)"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /mobilenumbers/{phoneNumber}/assign [post]
// @Security BearerAuth
func (h *MobileNumberHandler) AssignMobileNumber(c *gin.Context) {
	phoneNumberStr := c.Param("phoneNumber") // 从 :phoneNumber 获取
	//不再需要解析为 uint
	// numberID, err := parseUint(idStr)
	// if err != nil {
	// 	utils.RespondAPIError(c, http.StatusBadRequest, "无效的手机号码ID格式", err.Error())
	// 	return
	// }

	var payload models.MobileNumberAssignPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	assignmentDate, err := time.Parse("2006-01-02", payload.AssignmentDate)
	if err != nil {
		utils.RespondAPIError(c, http.StatusBadRequest, "分配日期格式无效，请使用 YYYY-MM-DD", err.Error())
		return
	}

	// 调用服务层，传递 phoneNumberStr 而不是 numberID
	assignedMobileNumber, err := h.service.AssignMobileNumber(phoneNumberStr, payload.EmployeeID, assignmentDate)
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
// @Description 校验目标号码是否为"在用"状态。更新号码记录，清空当前使用人员工ID，将号码状态改为"闲置"。更新上一条与该号码和使用人相关的号码使用历史记录，记录使用结束时间。
// @Tags MobileNumbers
// @Accept json
// @Produce json
// @Param phoneNumber path string true "手机号码字符串"
// @Param unassignPayload body models.MobileNumberUnassignPayload false "回收信息 (可选，包含回收日期 YYYY-MM-DD)"
// @Success 200 {object} utils.SuccessResponse{data=models.MobileNumber} "成功回收后的号码对象"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误 / 无效的日期格式"
// @Failure 401 {object} utils.APIErrorResponse "未认证或 Token 无效/过期"
// @Failure 404 {object} utils.APIErrorResponse "手机号码未找到"
// @Failure 409 {object} utils.APIErrorResponse "操作冲突 (例如：号码非在用状态，或未找到有效的分配记录)"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /mobilenumbers/{phoneNumber}/unassign [post]
// @Security BearerAuth
func (h *MobileNumberHandler) UnassignMobileNumber(c *gin.Context) {
	phoneNumberStr := c.Param("phoneNumber") // 读取 phoneNumber 字符串
	//不再需要 parseUint
	// numberID, err := parseUint(idStr)
	// if err != nil {
	// 	utils.RespondAPIError(c, http.StatusBadRequest, "无效的手机号码ID格式", err.Error())
	// 	return
	// }

	var payload models.MobileNumberUnassignPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	reclaimDate := time.Now()
	if payload.ReclaimDate != "" {
		parsedDate, err := time.Parse("2006-01-02", payload.ReclaimDate)
		if err != nil {
			utils.RespondAPIError(c, http.StatusBadRequest, "回收日期格式无效，请使用 YYYY-MM-DD", err.Error())
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
		case errors.Is(err, repositories.ErrMobileNumberNotInUseStatus):
			utils.RespondAPIError(c, http.StatusConflict, "手机号码不是在用状态，无法回收", err.Error())
		case errors.Is(err, repositories.ErrNoActiveUsageHistoryFound):
			utils.RespondAPIError(c, http.StatusConflict, "未找到该号码当前有效的分配记录，无法回收", err.Error())
		case strings.Contains(err.Error(), "数据不一致：在用号码没有关联当前用户"):
			utils.RespondAPIError(c, http.StatusInternalServerError, "服务器内部错误: 数据不一致", err.Error())
		default:
			utils.RespondInternalServerError(c, "回收手机号码失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusOK, unassignedMobileNumber, "手机号码回收成功")
}
