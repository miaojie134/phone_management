package handlers

import (
	"fmt"
	"net/http"
	"navigator/internal/services"
	"strconv"

	// "navigator/pkg/utils" // Assuming a response utility package

	"github.com/gin-gonic/gin"
)

// VerificationHandler handles HTTP requests for verification processes.
type VerificationHandler struct {
	service services.VerificationService
}

// NewVerificationHandler creates a new instance of VerificationHandler.
func NewVerificationHandler(service services.VerificationService) *VerificationHandler {
	return &VerificationHandler{service: service}
}

// --- Request/Response DTOs specific to handlers ---

// InitiateRequest is the DTO for the POST /initiate endpoint.
type InitiateRequest struct {
	Scope         string   `json:"scope" binding:"required"` // "all_users", "department_ids", "employee_ids"
	ScopeDetail   []string `json:"scopeDetail"`            // List of department or employee IDs if applicable
	DurationDays  *int     `json:"durationDays,omitempty"` // Optional, positive integer
}

// SubmitVerificationRequestDTO is the DTO for the POST /submit endpoint.
// Note: services.SubmitVerificationRequest is already defined, we might reuse or adapt.
// For now, let's assume the service layer DTO is suitable or we'll map.
// This DTO is specifically for binding JSON in the handler.
type SubmitVerificationRequestDTO struct {
	VerifiedNumbers         []services.VerifiedNumberAction       `json:"verifiedNumbers"`
	UnlistedNumbersReported []services.UnlistedNumberReportAction `json:"unlistedNumbersReported"`
}


// --- Handler Implementations ---

// InitiateVerificationHandler handles POST /api/v1/verification/initiate
func (h *VerificationHandler) InitiateVerificationHandler(c *gin.Context) {
	var req InitiateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}

	if req.DurationDays != nil && *req.DurationDays <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "durationDays must be positive if provided"})
		return
	}

	// Basic validation for scope and scopeDetail
	if req.Scope == "department_ids" || req.Scope == "employee_ids" {
		if len(req.ScopeDetail) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("scopeDetail is required when scope is '%s'", req.Scope)})
			return
		}
	} else if req.Scope == "all_users" && len(req.ScopeDetail) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scopeDetail should be empty when scope is 'all_users'"})
		return
	}


	count, err := h.service.InitiateVerificationProcess(req.Scope, req.ScopeDetail, req.DurationDays)
	if err != nil {
		// TODO: Differentiate service errors (e.g., validation vs. internal)
		// For now, a generic 500 for service errors, or 400 if it's clearly a validation issue.
		// This error check could be more sophisticated based on error types returned by the service.
		if err.Error() == "no employees found for the given scope" || 
		   (len(err.Error()) > 14 && err.Error()[:14] == "invalid scope:") ||
		   (len(err.Error()) > 20 && err.Error()[:20] == "invalid department ID") ||
		   (len(err.Error()) > 19 && err.Error()[:19] == "invalid employee ID") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate verification: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        fmt.Sprintf("Successfully initiated verification for %d employee(s).", count),
		"initiatedCount": count,
	})
}

// GetVerificationInfoHandler handles GET /api/v1/verification/info?token=<token>
func (h *VerificationHandler) GetVerificationInfoHandler(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token query parameter is required"})
		return
	}

	info, err := h.service.GetVerificationInfo(tokenStr)
	if err != nil {
		// "verification token not found", "verification token is already %s", "verification token has expired"
		// "error fetching employee details", "employee associated with token not found"
		// These could be mapped to 403/404
		// TODO: Use typed errors from service layer for better distinction
		errMsg := err.Error()
		if errMsg == "verification token not found" || 
		   errMsg == "verification token has expired" || 
		   (len(errMsg) > 29 && errMsg[:29] == "verification token is already") ||
		   errMsg == "employee associated with token not found" {
			c.JSON(http.StatusForbidden, gin.H{"error": "无效或已过期的链接。"}) 
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving verification info: " + errMsg})
		}
		return
	}

	c.JSON(http.StatusOK, info)
}

// SubmitVerificationHandler handles POST /api/v1/verification/submit?token=<token>
func (h *VerificationHandler) SubmitVerificationHandler(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token query parameter is required"})
		return
	}

	var reqDTO SubmitVerificationRequestDTO
	if err := c.ShouldBindJSON(&reqDTO); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	
	// Convert DTO to service layer request struct
	serviceReq := services.SubmitVerificationRequest{
		TokenStr:                tokenStr, // Token comes from query param, add it to service request
		VerifiedNumbers:         reqDTO.VerifiedNumbers,
		UnlistedNumbersReported: reqDTO.UnlistedNumbersReported,
	}


	err := h.service.SubmitVerification(serviceReq)
	if err != nil {
		errMsg := err.Error()
		// "verification token not found", "verification token is already %s", "verification token has expired"
		if errMsg == "verification token not found" ||
		   errMsg == "verification token has expired" ||
		   (len(errMsg) > 29 && errMsg[:29] == "verification token is already") {
			c.JSON(http.StatusForbidden, gin.H{"error": "无效或已过期的链接，或已提交过。"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit verification: " + errMsg})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "您的反馈已成功提交，感谢您的配合！"})
}

// GetAdminVerificationStatusHandler handles GET /api/v1/verification/admin/status
func (h *VerificationHandler) GetAdminVerificationStatusHandler(c *gin.Context) {
	var filters services.AdminStatusFilters

	if cycleIDStr := c.Query("cycleId"); cycleIDStr != "" {
		id, err := strconv.ParseUint(cycleIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cycleId format"})
			return
		}
		uid := uint(id)
		filters.CycleID = &uid
	}

	filters.Status = c.Query("status") // e.g., "pending", "used", "expired"

	if employeeIDStr := c.Query("employeeId"); employeeIDStr != "" {
		id, err := strconv.ParseUint(employeeIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employeeId format"})
			return
		}
		uid := uint(id)
		filters.EmployeeID = &uid
	}
	
	if deptIDStr := c.Query("departmentId"); deptIDStr != "" {
		id, err := strconv.ParseUint(deptIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid departmentId format"})
			return
		}
		uid := uint(id)
		filters.DepartmentID = &uid
	}


	statusData, err := h.service.GetAdminVerificationStatus(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve admin verification status: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, statusData)
}
