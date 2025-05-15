package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phone_management/internal/models"
	"github.com/phone_management/internal/repositories" // 用于判断 ErrPhoneNumberExists
	"github.com/phone_management/internal/services"
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
// @Success 201 {object} models.MobileNumber "创建成功的号码对象"
// @Failure 400 {object} map[string]interface{} "请求参数错误或数据校验失败 (e.g. {\"error\": \"请求参数无效\", \"details\": \"...\"})"
// @Failure 401 {object} map[string]interface{} "未认证或 Token 无效/过期 (e.g. {\"error\": \"未授权\"})"
// @Failure 409 {object} map[string]interface{} "手机号码已存在 (e.g. {\"error\": \"手机号码已存在\"})"
// @Failure 500 {object} map[string]interface{} "服务器内部错误 (e.g. {\"error\": \"创建手机号码失败\", \"details\": \"...\"})"
// @Router /mobilenumbers [post]
// @Security BearerAuth
func (h *MobileNumberHandler) CreateMobileNumber(c *gin.Context) {
	var payload CreateMobileNumberPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效", "details": err.Error()})
		return
	}

	// 解析 applicationDate 字符串为 time.Time
	applicationDate, err := time.Parse("2006-01-02", payload.ApplicationDate)
	if err != nil {
		// 这个错误理论上不应该发生，因为 Gin 的 datetime tag 已经校验过格式了
		// 但作为健壮性考虑，还是处理一下
		c.JSON(http.StatusBadRequest, gin.H{"error": "申请日期格式无效", "details": err.Error()})
		return
	}

	// 创建 models.MobileNumber 实例用于传递给 service 层
	mobileNumberToCreate := &models.MobileNumber{
		PhoneNumber:           payload.PhoneNumber,
		ApplicantEmployeeDbID: payload.ApplicantEmployeeDbID,
		ApplicationDate:       applicationDate,
		Status:                payload.Status,
		Vendor:                payload.Vendor,
		Remarks:               payload.Remarks,
		// 如果 payload 中有 CurrentEmployeeDbID 和 CancellationDate，也需要相应处理
		// CurrentEmployeeDbID: payload.CurrentEmployeeDbID,
	}

	// 如果 CancellationDate 在 payload 中是可选的，并且被提供了，则解析它
	// if payload.CancellationDate != "" {
	// 	cancellationDate, err := time.Parse("2006-01-02", payload.CancellationDate)
	// 	if err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": "注销日期格式无效", "details": err.Error()})
	// 		return
	// 	}
	// 	mobileNumberToCreate.CancellationDate = &cancellationDate
	// }

	createdMobileNumber, err := h.service.CreateMobileNumber(mobileNumberToCreate)
	if err != nil {
		if errors.Is(err, repositories.ErrPhoneNumberExists) {
			c.JSON(http.StatusConflict, gin.H{"error": repositories.ErrPhoneNumberExists.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建手机号码失败", "details": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, createdMobileNumber)
}
