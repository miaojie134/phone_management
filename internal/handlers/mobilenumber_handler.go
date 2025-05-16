package handlers

import (
	"errors"
	"net/http"
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
	PhoneNumber           string `json:"phoneNumber" binding:"required,max=50"`
	ApplicantEmployeeDbID uint   `json:"applicantEmployeeId" binding:"required"`
	ApplicationDate       string `json:"applicationDate" binding:"required,datetime=2006-01-02"` // 日期作为字符串接收和验证
	Status                string `json:"status" binding:"required,oneof=闲置 在用 待注销 已注销 待核实-办卡人离职"`
	Vendor                string `json:"vendor" binding:"max=100"`
	Remarks               string `json:"remarks" binding:"max=255"`
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
		PhoneNumber:           payload.PhoneNumber,
		ApplicantEmployeeDbID: payload.ApplicantEmployeeDbID,
		ApplicationDate:       applicationDate,
		Status:                payload.Status,
		Vendor:                payload.Vendor,
		Remarks:               payload.Remarks,
	}

	createdMobileNumber, err := h.service.CreateMobileNumber(mobileNumberToCreate)
	if err != nil {
		if errors.Is(err, repositories.ErrPhoneNumberExists) {
			utils.RespondConflictError(c, repositories.ErrPhoneNumberExists.Error())
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

type PaginationInfo struct {
	TotalItems  int64 `json:"totalItems"`
	TotalPages  int64 `json:"totalPages"`
	CurrentPage int   `json:"currentPage"`
	PageSize    int   `json:"pageSize"`
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
