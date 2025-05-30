package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/phone_management/internal/models"
	"github.com/phone_management/internal/services"
	"github.com/phone_management/pkg/utils"
)

// VerificationHandler 负责处理与号码验证流程相关的 HTTP 请求
type VerificationHandler struct {
	verificationService services.VerificationService
}

// NewVerificationHandler 创建一个新的 VerificationHandler 实例
func NewVerificationHandler(vs services.VerificationService) *VerificationHandler {
	return &VerificationHandler{verificationService: vs}
}

// InitiateVerificationRequest 定义了发起确认流程 API 的请求体结构
type InitiateVerificationRequest struct {
	Scope        string   `json:"scope" binding:"required,oneof=all_users department employee_ids"`
	ScopeValues  []string `json:"scopeValues,omitempty"`
	DurationDays int      `json:"durationDays" binding:"required,min=1,max=30"`
}

// InitiateVerificationResponse 定义了发起确认流程API成功时的响应体
type InitiateVerificationResponse struct {
	BatchID string `json:"batchId"`
}

// GetVerificationInfo godoc
// @Summary 获取待确认的号码信息
// @Description 用户点击邮件链接后，前端页面调用此接口获取该用户需确认的号码信息
// @Tags Verification
// @Produce json
// @Param token query string true "验证令牌"
// @Success 200 {object} utils.SuccessResponse{data=models.VerificationInfo} "成功响应，包含员工姓名、令牌有效期、待验证的号码列表及其状态"
// @Failure 403 {object} utils.APIErrorResponse "令牌无效或已过期"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /verification/info [get]
func (h *VerificationHandler) GetVerificationInfo(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		utils.RespondAPIError(c, http.StatusBadRequest, "请求参数无效", "缺少token参数")
		return
	}

	info, err := h.verificationService.GetVerificationInfo(c.Request.Context(), token)
	if err != nil {
		switch err {
		case services.ErrTokenNotFound, services.ErrTokenExpired:
			utils.RespondAPIError(c, http.StatusForbidden, "无效或已过期的链接。", err.Error())
		default:
			utils.RespondInternalServerError(c, "获取验证信息失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusOK, info, "成功获取待确认号码信息")
}

// InitiateVerification godoc
// @Summary 发起号码使用确认流程 (异步)
// @Description 管理员调用此接口后，系统创建一个批处理任务来为目标员工生成 VerificationTokens 并发送邮件。接口立即返回批处理ID。
// @Tags Verification
// @Accept json
// @Produce json
// @Param body body InitiateVerificationRequest true "请求体"
// @Success 202 {object} utils.SuccessResponse{data=InitiateVerificationResponse} "请求已接受，批处理任务已创建"
// @Failure 400 {object} utils.APIErrorResponse "请求参数无效或错误"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Security BearerAuth
// @Router /verification/initiate [post]
func (h *VerificationHandler) InitiateVerification(c *gin.Context) {
	var req InitiateVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	if (req.Scope == string(models.VerificationScopeDepartment) || req.Scope == string(models.VerificationScopeEmployeeIDs)) && len(req.ScopeValues) == 0 {
		utils.RespondAPIError(c, http.StatusBadRequest, fmt.Sprintf("当 scope 为 %s 或 %s 时，scopeValues 不能为空", models.VerificationScopeDepartment, models.VerificationScopeEmployeeIDs), nil)
		return
	}

	var scopeType models.VerificationScopeType
	switch req.Scope {
	case string(models.VerificationScopeAllUsers):
		scopeType = models.VerificationScopeAllUsers
	case string(models.VerificationScopeDepartment):
		scopeType = models.VerificationScopeDepartment
	case string(models.VerificationScopeEmployeeIDs):
		scopeType = models.VerificationScopeEmployeeIDs
	default:
		utils.RespondAPIError(c, http.StatusBadRequest, fmt.Sprintf("无效的 scope 类型: %s", req.Scope), nil)
		return
	}

	batchID, err := h.verificationService.InitiateVerificationProcess(c.Request.Context(), scopeType, req.ScopeValues, req.DurationDays)
	if err != nil {
		// 根据错误类型，可能返回不同的状态码，例如，如果是预检员工查找失败，可能是400或500
		// 暂时统一处理为500
		utils.RespondInternalServerError(c, "发起确认流程失败", err.Error())
		return
	}

	responseData := InitiateVerificationResponse{
		BatchID: batchID,
	}
	utils.RespondSuccess(c, http.StatusAccepted, responseData, "号码确认流程已作为批处理任务启动。")
}

// GetVerificationBatchStatus godoc
// @Summary 获取号码确认批处理任务的状态
// @Description 获取指定号码确认批处理任务的当前状态、整体进度（包括已处理员工数、令牌生成情况、邮件发送统计：尝试数、成功数、失败数）以及详细的错误报告（例如邮件发送失败的原因）。
// @Tags Verification
// @Produce json
// @Param batchId path string true "批处理任务ID"
// @Success 200 {object} utils.SuccessResponse{data=models.VerificationBatchTask} "成功响应，包含批处理任务详情"
// @Failure 400 {object} utils.APIErrorResponse "无效的批处理ID格式"
// @Failure 404 {object} utils.APIErrorResponse "批处理任务未找到"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Security BearerAuth
// @Router /verification/batch/{batchId}/status [get]
func (h *VerificationHandler) GetVerificationBatchStatus(c *gin.Context) {
	batchID := c.Param("batchId")
	if batchID == "" { // 基本校验，更严格的 UUID 校验可以在服务层或模型层
		utils.RespondAPIError(c, http.StatusBadRequest, "批处理ID不能为空", nil)
		return
	}

	task, err := h.verificationService.GetVerificationBatchStatus(c.Request.Context(), batchID)
	if err != nil {
		if err == services.ErrBatchTaskNotFound { // 使用在 service 层定义的错误
			utils.RespondAPIError(c, http.StatusNotFound, fmt.Sprintf("批处理任务 %s 未找到", batchID), err.Error())
		} else {
			utils.RespondInternalServerError(c, "获取批处理任务状态失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusOK, task, "成功获取批处理任务状态。")
}

// SubmitVerificationResult godoc
// @Summary 提交号码确认结果
// @Description 用户提交其号码确认结果，包括"确认使用"或"报告问题"的号码，以及可能上报的未在系统中列出但实际在使用的号码
// @Tags Verification
// @Accept json
// @Produce json
// @Param token query string true "验证令牌 - 从邮件链接中获取的token参数"
// @Param body body models.VerificationSubmission true "请求体，包含 verifiedNumbers（必填，号码ID、动作类型、用途purpose、可选的备注）和 unlistedNumbersReported（可选，用户报告的未列出号码，需包含phoneNumber和必填的purpose）"
// @Success 200 {object} utils.SuccessResponse "提交成功"
// @Failure 400 {object} utils.APIErrorResponse "请求参数无效"
// @Failure 403 {object} utils.APIErrorResponse "令牌无效或已过期"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /verification/submit [post]
func (h *VerificationHandler) SubmitVerificationResult(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		utils.RespondAPIError(c, http.StatusBadRequest, "请求参数无效", "缺少token参数")
		return
	}

	var req models.VerificationSubmission
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondValidationError(c, err.Error())
		return
	}

	// 请求参数基本验证：VerifiedNumbers 和 UnlistedNumbersReported 至少要有一个
	if len(req.VerifiedNumbers) == 0 && len(req.UnlistedNumbersReported) == 0 {
		utils.RespondAPIError(c, http.StatusBadRequest, "请求参数无效", "verifiedNumbers 和 unlistedNumbersReported 不能同时为空")
		return
	}

	// 校验 UserComment for VerifiedNumbers
	for _, vn := range req.VerifiedNumbers {
		if vn.Action == "report_issue" {
			trimmedComment := strings.TrimSpace(vn.UserComment)
			if trimmedComment == "" {
				errorMessage := fmt.Sprintf("当操作为 'report_issue' 时，号码ID %d 的用户备注 (userComment) 不能为空。", vn.MobileNumberId)
				utils.RespondAPIError(c, http.StatusBadRequest, "请求参数无效", errorMessage)
				return
			}
			if len(trimmedComment) > 500 { // 假设最大长度为500
				errorMessage := fmt.Sprintf("号码ID %d 的用户备注过长，请保持在500字符以内。", vn.MobileNumberId)
				utils.RespondAPIError(c, http.StatusBadRequest, "请求参数无效", errorMessage)
				return
			}
			// 如果需要，可以将TrimmedComment赋值回vn.UserComment，但这取决于后续服务层是否期望处理首尾空格
			// vn.UserComment = trimmedComment
		}

		// 验证 Purpose 字段（如果提供）-> 修改为必填
		if vn.Purpose == nil || strings.TrimSpace(*vn.Purpose) == "" {
			errorMessage := fmt.Sprintf("号码ID %d 的用途 (purpose) 不能为空。", vn.MobileNumberId)
			utils.RespondAPIError(c, http.StatusBadRequest, "请求参数无效", errorMessage)
			return
		}
		if len(*vn.Purpose) > 255 { // 限制用途字段长度
			errorMessage := fmt.Sprintf("号码ID %d 的用途描述过长，请保持在255字符以内。", vn.MobileNumberId)
			utils.RespondAPIError(c, http.StatusBadRequest, "请求参数无效", errorMessage)
			return
		}
	}

	// 校验 UserComment for UnlistedNumbersReported (如果需要，可以添加类似校验)
	// 例如：如果 unlistedNumbersReported 中的 userComment 也需要校验
	for _, un := range req.UnlistedNumbersReported {
		// 验证 Purpose 字段 (必填)
		if un.Purpose == nil || strings.TrimSpace(*un.Purpose) == "" {
			errorMessage := fmt.Sprintf("报告的未列出号码 %s 的用途 (purpose) 不能为空。", un.PhoneNumber)
			utils.RespondAPIError(c, http.StatusBadRequest, "请求参数无效", errorMessage)
			return
		}
		if len(*un.Purpose) > 255 { // 限制用途字段长度
			errorMessage := fmt.Sprintf("报告的未列出号码 %s 的用途描述过长，请保持在255字符以内。", un.PhoneNumber)
			utils.RespondAPIError(c, http.StatusBadRequest, "请求参数无效", errorMessage)
			return
		}

		trimmedComment := strings.TrimSpace(un.UserComment)
		if len(trimmedComment) > 500 { // 假设最大长度为500
			errorMessage := fmt.Sprintf("报告的未列出号码 %s 的用户备注过长，请保持在500字符以内。", un.PhoneNumber)
			utils.RespondAPIError(c, http.StatusBadRequest, "请求参数无效", errorMessage)
			return
		}
		// 如果 UnlistedNumbersReported 的 UserComment 在某些情况下也是必填的，可以在这里添加校验
		// if trimmedComment == "" {
		// 	 utils.RespondAPIError(c, http.StatusBadRequest, "请求参数无效", fmt.Sprintf("报告的未列出号码 %s 的用户备注不能为空。", un.PhoneNumber))
		// 	 return
		// }
	}

	// 处理确认结果，直接传递models中的结构体
	err := h.verificationService.SubmitVerificationResult(c.Request.Context(), token, &req)
	if err != nil {
		switch err {
		case services.ErrTokenNotFound, services.ErrTokenExpired:
			utils.RespondAPIError(c, http.StatusForbidden, "无效或已过期的链接。", err.Error())
		default:
			utils.RespondInternalServerError(c, "提交确认结果失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusOK, nil, "您的反馈已成功提交，感谢您的配合！")
}

/*
// GetVerificationAdminStatus godoc
// @Summary 获取管理员视图的号码确认流程状态
// @Description 获取管理员视图的号码确认流程状态，包括统计摘要和详细信息
// @Tags Verification
// @Accept json
// @Produce json
// @Param employee_id query string false "员工业务工号，用于筛选"
// @Param employeeId query string false "员工业务工号，用于筛选（兼容旧版本）"
// @Param department query string false "部门名称，用于筛选"
// @Param departmentName query string false "部门名称，用于筛选（兼容旧版本）"
// @Success 200 {object} utils.SuccessResponse{data=models.AdminVerificationStatusResponse} "成功响应"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误"
// @Failure 401 {object} utils.APIErrorResponse "未授权"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /verification/admin/status [get]
// @Security BearerAuth
func (h *VerificationHandler) GetVerificationAdminStatus(c *gin.Context) {
	// 从查询参数中获取可选的筛选条件
	// 优先使用标准化的参数名（带下划线），如果不存在则使用驼峰式命名参数（兼容旧版本）
	employeeID := c.Query("employee_id")
	if employeeID == "" {
		employeeID = c.Query("employeeId")
	}

	departmentName := c.Query("department")
	if departmentName == "" {
		departmentName = c.Query("departmentName")
	}

	// 调用服务层获取管理员视图的号码确认流程状态
	adminStatus, err := h.verificationService.GetAdminVerificationStatus(c.Request.Context(), employeeID, departmentName)
	if err != nil {
		utils.RespondInternalServerError(c, "获取管理员视图的号码确认流程状态失败", err.Error())
		return
	}

	utils.RespondSuccess(c, http.StatusOK, adminStatus, "获取管理员视图的号码确认流程状态成功")
}
*/

// GetPhoneVerificationStatus godoc
// @Summary 获取基于手机号码维度的确认流程状态
// @Description 获取基于手机号码维度的确认流程状态，包括统计摘要和详细信息
// @Tags Verification
// @Accept json
// @Produce json
// @Param employee_id query string false "员工业务工号，用于筛选"
// @Param employeeId query string false "员工业务工号，用于筛选（兼容旧版本）"
// @Param department query string false "部门名称，用于筛选"
// @Param departmentName query string false "部门名称，用于筛选（兼容旧版本）"
// @Success 200 {object} utils.SuccessResponse{data=models.PhoneVerificationStatusResponse} "成功响应"
// @Failure 400 {object} utils.APIErrorResponse "请求参数错误"
// @Failure 401 {object} utils.APIErrorResponse "未授权"
// @Failure 500 {object} utils.APIErrorResponse "服务器内部错误"
// @Router /verification/admin/phone-status [get]
// @Security BearerAuth
func (h *VerificationHandler) GetPhoneVerificationStatus(c *gin.Context) {
	// 从查询参数中获取可选的筛选条件
	// 优先使用标准化的参数名（带下划线），如果不存在则使用驼峰式命名参数（兼容旧版本）
	employeeID := c.Query("employee_id")
	if employeeID == "" {
		employeeID = c.Query("employeeId")
	}

	departmentName := c.Query("department")
	if departmentName == "" {
		departmentName = c.Query("departmentName")
	}

	// 调用服务层获取基于手机号码维度的确认流程状态
	phoneStatus, err := h.verificationService.GetPhoneVerificationStatus(c.Request.Context(), employeeID, departmentName)
	if err != nil {
		utils.RespondInternalServerError(c, "获取基于手机号码维度的确认流程状态失败", err.Error())
		return
	}

	utils.RespondSuccess(c, http.StatusOK, phoneStatus, "获取基于手机号码维度的确认流程状态成功")
}
