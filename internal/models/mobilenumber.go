package models

import (
	"time"

	"gorm.io/gorm"
)

// NumberStatus 定义了号码状态的枚举
type NumberStatus string

const (
	StatusIdle                NumberStatus = "闲置"
	StatusInUse               NumberStatus = "在用"
	StatusPendingDeactivation NumberStatus = "待注销"
	StatusDeactivated         NumberStatus = "已注销"
	StatusRiskPending         NumberStatus = "待核实-办卡人离职"
	StatusUserReport          NumberStatus = "待核实-用户报告"
)

// MobileNumber 对应于数据库中的 mobile_numbers 表
type MobileNumber struct {
	ID                   uint           `json:"id" gorm:"primaryKey"`
	PhoneNumber          string         `json:"phoneNumber" gorm:"unique;not null;size:11" binding:"required,len=11,numeric"`
	ApplicantEmployeeID  string         `json:"applicantEmployeeId" gorm:"column:applicant_employee_id;not null" binding:"required"` // 办卡人员工业务工号
	ApplicationDate      time.Time      `json:"applicationDate" gorm:"not null" binding:"required,time_format=2006-01-02"`
	CurrentEmployeeID    *string        `json:"currentEmployeeId,omitempty" gorm:"column:current_employee_id"` // 当前使用人员工业务工号
	Status               string         `json:"status" gorm:"not null" binding:"required,oneof=闲置 在用 待注销 已注销 待核实-办卡人离职 待核实-用户报告"`
	Purpose              *string        `json:"purpose,omitempty" gorm:"type:varchar(255);null"` // 号码用途，例如"办公"、"客户联系"等
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
	Status              string               `json:"status"`
	Purpose             *string              `json:"purpose,omitempty"` // 号码用途
	Vendor              string               `json:"vendor,omitempty"`
	Remarks             string               `json:"remarks,omitempty"`
	CancellationDate    *time.Time           `json:"cancellationDate,omitempty"`
	CreatedAt           time.Time            `json:"createdAt"`
	UpdatedAt           time.Time            `json:"updatedAt"`
	UsageHistory        []NumberUsageHistory `json:"usageHistory,omitempty" gorm:"foreignKey:MobileNumberDbID"` // 号码使用历史
}

// MobileNumberUpdatePayload 定义了更新手机号码信息的请求体结构
type MobileNumberUpdatePayload struct {
	Status  *string `json:"status,omitempty" binding:"omitempty,oneof=闲置 在用 待注销 已注销 待核实-办卡人离职 待核实-用户报告"`
	Purpose *string `json:"purpose,omitempty" binding:"omitempty,max=255"`
	Vendor  *string `json:"vendor,omitempty" binding:"omitempty,max=100"`
	Remarks *string `json:"remarks,omitempty" binding:"omitempty,max=255"`
}

// MobileNumberAssignPayload 定义了分配号码的请求体
type MobileNumberAssignPayload struct {
	EmployeeID     string `json:"employeeId" binding:"required"` // 员工业务工号
	AssignmentDate string `json:"assignmentDate" binding:"required,datetime=2006-01-02"`
}

// MobileNumberUnassignPayload 定义了回收号码的请求体
type MobileNumberUnassignPayload struct {
	ReclaimDate string `json:"reclaimDate,omitempty" binding:"omitempty,datetime=2006-01-02"`
}

// MobileNumberBasicInfo 用于员工详情中展示的号码简要信息
type MobileNumberBasicInfo struct {
	ID          uint   `json:"id"`
	PhoneNumber string `json:"phoneNumber"`
	Status      string `json:"status"`
}
