package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SuccessResponse 定义了标准的成功响应结构
type SuccessResponse struct {
	Status  string      `json:"status"`            // 例如 "success"
	Message string      `json:"message,omitempty"` // 可选的成功消息
	Data    interface{} `json:"data,omitempty"`    // 响应数据
}

// ErrorResponse 定义了标准的错误响应结构
type ErrorResponse struct {
	Status  string   `json:"status"`            // 例如 "error"
	Message string   `json:"message"`           // 错误信息
	Details []string `json:"details,omitempty"` // 可选的错误详情
}

// RespondJSON 是一个通用的辅助函数，用于发送 JSON 响应
func RespondJSON(c *gin.Context, status int, payload interface{}) {
	c.JSON(status, payload)
}

// RespondSuccess 发送一个标准的成功 JSON 响应
// status: HTTP 状态码 (例如 http.StatusOK, http.StatusCreated)
// data: 要包含在响应中的数据
// message: (可选) 成功消息
func RespondSuccess(c *gin.Context, status int, data interface{}, message string) {
	response := SuccessResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	}
	if message == "" && data == nil { // 如果没有消息和数据，确保结构仍然合理
		response.Message = "Operation successful"
	}
	RespondJSON(c, status, response)
}

// RespondError 发送一个标准的错误 JSON 响应
// status: HTTP 状态码 (例如 http.StatusBadRequest, http.StatusInternalServerError)
// message: 主要的错误信息
// details: (可选) 额外的错误详情
func RespondError(c *gin.Context, status int, message string, details ...string) {
	response := ErrorResponse{
		Status:  "error",
		Message: message,
	}
	if len(details) > 0 {
		response.Details = details
	}
	RespondJSON(c, status, response)
}

// RespondGinError 是一个辅助函数，用于处理 gin 绑定错误或其他通用错误
// 它将 error.Error() 作为主要消息
func RespondGinError(c *gin.Context, status int, err error, defaultMessage ...string) {
	msg := err.Error()
	if len(defaultMessage) > 0 && defaultMessage[0] != "" {
		msg = defaultMessage[0]
	}
	RespondError(c, status, msg, err.Error())
}

// --- 以下是根据 API 文档中建议的特定错误格式的辅助函数 ---

// APIErrorResponse 对应 API 文档中的错误响应格式 { "error": "描述信息", "details": { ... } }
// 注意: details 可以是 map[string]interface{} 或 string
type APIErrorResponse struct {
	Error   string      `json:"error"`
	Details interface{} `json:"details,omitempty"`
}

// RespondAPIError 发送符合特定 API 文档格式的错误响应
func RespondAPIError(c *gin.Context, status int, errorMessage string, details interface{}) {
	response := APIErrorResponse{
		Error: errorMessage,
	}
	if details != nil {
		response.Details = details
	}
	c.AbortWithStatusJSON(status, response)
}

// RespondValidationError 发送用于处理参数校验错误的特定响应
// details 通常是 err.Error() 或更结构化的错误信息
func RespondValidationError(c *gin.Context, details interface{}) {
	RespondAPIError(c, http.StatusBadRequest, "请求参数无效", details)
}

// RespondUnauthorizedError 发送未授权错误
func RespondUnauthorizedError(c *gin.Context, message ...string) {
	errMsg := "未认证或 Token 无效/过期"
	if len(message) > 0 && message[0] != "" {
		errMsg = message[0]
	}
	RespondAPIError(c, http.StatusUnauthorized, errMsg, nil)
}

// RespondNotFoundError 发送资源未找到错误
func RespondNotFoundError(c *gin.Context, resourceName string) {
	RespondAPIError(c, http.StatusNotFound, resourceName+"未找到", nil)
}

// RespondInternalServerError 发送服务器内部错误
// errDetails 可以是 err.Error()
func RespondInternalServerError(c *gin.Context, message string, errDetails ...string) {
	var details interface{}
	if len(errDetails) > 0 {
		details = errDetails[0]
	}
	RespondAPIError(c, http.StatusInternalServerError, message, details)
}

// RespondConflictError 发送冲突错误 (例如，资源已存在)
func RespondConflictError(c *gin.Context, message string, details ...string) {
	var detailContent interface{}
	if len(details) > 0 {
		detailContent = details[0]
	}
	RespondAPIError(c, http.StatusConflict, message, detailContent)
}
