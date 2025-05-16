package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

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

// CreateEmployeePayload 定义了创建员工请求的 JSON 结构体
type CreateEmployeePayload struct {
	EmployeeID string  `json:"employeeId" binding:"required,max=100"`
	FullName   string  `json:"fullName" binding:"required,max=255"`
	Department *string `json:"department,omitempty" binding:"omitempty,max=255"`
	// EmploymentStatus 默认为 "Active"，在模型或服务层处理，此处不需传递
}

// PagedEmployeesData 定义了员工列表的分页响应结构
type PagedEmployeesData struct {
	Items      []models.Employee `json:"items"`
	Pagination PaginationInfo    `json:"pagination"`
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
	var payload CreateEmployeePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	employeeToCreate := &models.Employee{
		EmployeeID: payload.EmployeeID,
		FullName:   payload.FullName,
		Department: payload.Department,
		// EmploymentStatus 将由服务层或模型默认设置
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

// parseUintFromString 是一个辅助函数，用于将字符串ID解析为uint
// TODO: 考虑将此类通用辅助函数移至共享的 utils 包中，如果多处需要
func parseUintFromString(idStr string) (uint, error) {
	val, err := strconv.ParseUint(idStr, 10, 32) // 32表示结果适合uint类型
	if err != nil {
		return 0, fmt.Errorf("无法将 '%s' 解析为有效的数字ID: %w", idStr, err)
	}
	return uint(val), nil
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
