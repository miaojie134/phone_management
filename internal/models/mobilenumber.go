package models

import (
	"time"

	"gorm.io/gorm"
)

// NumberStatus 定义了号码状态的枚举
type NumberStatus string

const (
	StatusIdle                NumberStatus = "idle"                 // 闲置
	StatusInUse               NumberStatus = "in_use"               // 使用中
	StatusPendingDeactivation NumberStatus = "pending_deactivation" // 待注销
	StatusDeactivated         NumberStatus = "deactivated"          // 已注销
	StatusRiskPending         NumberStatus = "risk_pending"         // 待核实-办卡人离职
	StatusUserReport          NumberStatus = "user_reported"        // 待核实-用户报告
)

// RiskHandleAction 定义了风险号码处理操作的枚举
type RiskHandleAction string

const (
	ActionChangeApplicant RiskHandleAction = "change_applicant" // 变更办卡人
	ActionReclaim         RiskHandleAction = "reclaim"          // 回收号码
	ActionDeactivate      RiskHandleAction = "deactivate"       // 注销号码
)

// GetAllStatuses 返回所有可用的状态
func GetAllStatuses() []NumberStatus {
	return []NumberStatus{
		StatusIdle,
		StatusInUse,
		StatusPendingDeactivation,
		StatusDeactivated,
		StatusRiskPending,
		StatusUserReport,
	}
}

// IsValidStatus 检查状态是否有效
func IsValidStatus(status string) bool {
	for _, validStatus := range GetAllStatuses() {
		if string(validStatus) == status {
			return true
		}
	}
	return false
}

// GetAllRiskHandleActions 返回所有可用的风险处理操作
func GetAllRiskHandleActions() []RiskHandleAction {
	return []RiskHandleAction{
		ActionChangeApplicant,
		ActionReclaim,
		ActionDeactivate,
	}
}

// IsValidRiskHandleAction 检查风险处理操作是否有效
func IsValidRiskHandleAction(action string) bool {
	for _, validAction := range GetAllRiskHandleActions() {
		if string(validAction) == action {
			return true
		}
	}
	return false
}

// MobileNumber 对应于数据库中的 mobile_numbers 表
type MobileNumber struct {
	ID                   uint           `json:"id" gorm:"primaryKey"`
	PhoneNumber          string         `json:"phoneNumber" gorm:"unique;not null;size:11" binding:"required,len=11,numeric"`
	ApplicantEmployeeID  string         `json:"applicantEmployeeId" gorm:"column:applicant_employee_id;not null" binding:"required"` // 办卡人员工业务工号
	ApplicationDate      time.Time      `json:"applicationDate" gorm:"not null" binding:"required,time_format=2006-01-02"`
	CurrentEmployeeID    *string        `json:"currentEmployeeId,omitempty" gorm:"column:current_employee_id"` // 当前使用人员工业务工号
	Status               string         `json:"status" gorm:"not null"`                                        // 使用英文常量存储
	Purpose              *string        `json:"purpose,omitempty" gorm:"type:varchar(255);null"`               // 号码用途，例如"办公"、"客户联系"等
	Vendor               string         `json:"vendor" binding:"max=100"`
	Remarks              string         `json:"remarks" binding:"max=255"`
	CancellationDate     *time.Time     `json:"cancellationDate" binding:"omitempty,time_format=2006-01-02"`
	LastConfirmationDate *time.Time     `json:"lastConfirmationDate" gorm:"column:last_confirmation_date"` // 最后确认日期
	CreatedAt            time.Time      `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt            time.Time      `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt            gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index" swaggertype:"string" format:"date-time"`
}

// TableName 指定 MobileNumber 结构体对应的数据库表名
func (MobileNumber) TableName() string {
	return "mobile_numbers"
}

// MobileNumberResponse 是用于 API 响应的手机号码数据结构，包含关联信息
type MobileNumberResponse struct {
	ID                  uint                 `json:"id"`
	PhoneNumber         string               `json:"phoneNumber"`
	ApplicantEmployeeID string               `json:"applicantEmployeeId"`       // 办卡人员工业务工号
	ApplicantName       string               `json:"applicantName,omitempty"`   // 办卡人姓名
	ApplicantStatus     string               `json:"applicantStatus,omitempty"` // 办卡人当前在职状态
	ApplicationDate     time.Time            `json:"applicationDate"`
	CurrentEmployeeID   *string              `json:"currentEmployeeId,omitempty"` // 当前使用人员工业务工号
	CurrentUserName     string               `json:"currentUserName,omitempty"`   // 当前使用人姓名
	Status              string               `json:"status"`                      // 英文状态值
	Purpose             *string              `json:"purpose,omitempty"`           // 号码用途
	Vendor              string               `json:"vendor,omitempty"`
	Remarks             string               `json:"remarks,omitempty"`
	CancellationDate    *time.Time           `json:"cancellationDate,omitempty"`
	CreatedAt           time.Time            `json:"createdAt"`
	UpdatedAt           time.Time            `json:"updatedAt"`
	UsageHistory        []NumberUsageHistory `json:"usageHistory,omitempty" gorm:"foreignKey:MobileNumberDbID"` // 号码使用历史
}

// MobileNumberUpdatePayload 定义了更新手机号码信息的请求体结构
type MobileNumberUpdatePayload struct {
	Status  *string `json:"status,omitempty"`
	Purpose *string `json:"purpose,omitempty" binding:"omitempty,max=255"`
	Vendor  *string `json:"vendor,omitempty" binding:"omitempty,max=100"`
	Remarks *string `json:"remarks,omitempty" binding:"omitempty,max=255"`
}

// MobileNumberAssignPayload 定义了分配号码的请求体
type MobileNumberAssignPayload struct {
	EmployeeID     string `json:"employeeId" binding:"required"` // 员工业务工号
	AssignmentDate string `json:"assignmentDate" binding:"required,datetime=2006-01-02"`
	Purpose        string `json:"purpose" binding:"required,max=255"` // 号码用途，必填
}

// MobileNumberUnassignPayload 定义了回收号码的请求体
type MobileNumberUnassignPayload struct {
	ReclaimDate string `json:"reclaimDate,omitempty" binding:"omitempty,datetime=2006-01-02"`
}

// MobileNumberBasicInfo 用于员工详情中展示的号码简要信息
type MobileNumberBasicInfo struct {
	ID          uint   `json:"id"`
	PhoneNumber string `json:"phoneNumber"`
	Status      string `json:"status"` // 英文状态值
}

// HandleRiskNumberPayload 定义了处理风险号码的请求体
type HandleRiskNumberPayload struct {
	Action                 string  `json:"action" binding:"required"`                            // 操作类型：变更办卡人、回收、注销
	NewApplicantEmployeeID *string `json:"newApplicantEmployeeId,omitempty" binding:"omitempty"` // 新办卡人员工业务工号（变更办卡人时必填）
	Remarks                string  `json:"remarks" binding:"omitempty,max=500"`                  // 备注
}

// RiskNumberResponse 风险号码响应结构，继承MobileNumberResponse并增加风险相关信息
type RiskNumberResponse struct {
	MobileNumberResponse
	ApplicantDepartureDate *time.Time `json:"applicantDepartureDate,omitempty"` // 办卡人离职日期
	DaysSinceDeparture     *int       `json:"daysSinceDeparture,omitempty"`     // 离职天数
}
