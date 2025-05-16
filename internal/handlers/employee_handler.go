package handlers

import (
	"errors"
	"net/http"

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
