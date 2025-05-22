package handlers

import (
	"fmt"
	"net/http"

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
// @Success 200 {object} utils.SuccessResponse{data=services.VerificationInfoResponse} "成功响应，包含员工姓名、令牌有效期、待验证的号码列表"
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
		case services.ErrTokenNotFound, services.ErrTokenExpired, services.ErrTokenUsed:
			utils.RespondAPIError(c, http.StatusForbidden, "无效或已过期的链接。", err.Error())
		default:
			utils.RespondInternalServerError(c, "获取验证信息失败", err.Error())
		}
		return
	}

	utils.RespondSuccess(c, http.StatusOK, info, "成功获取待确认号码信息")
}

// GetBatchStatusResponse 定义了获取批处理任务状态 API 的响应体 (直接使用 models.VerificationBatchTask 即可)
// type GetBatchStatusResponse struct {
// 	 models.VerificationBatchTask
// }

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
