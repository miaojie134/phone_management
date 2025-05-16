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
)

// MobileNumber 对应于数据库中的 mobile_numbers 表
type MobileNumber struct {
	ID                    uint           `json:"id" gorm:"primaryKey"`
	PhoneNumber           string         `json:"phoneNumber" gorm:"unique;not null" binding:"required,max=50"`
	ApplicantEmployeeDbID uint           `json:"applicantEmployeeId" gorm:"not null" binding:"required"`                    // json 标签改为 applicantEmployeeId 以匹配API请求体
	ApplicationDate       time.Time      `json:"applicationDate" gorm:"not null" binding:"required,time_format=2006-01-02"` // 修正：time_format 合并到 binding 标签
	CurrentEmployeeDbID   *uint          `json:"currentEmployeeDbId"`                                                       // 当前使用人员工记录的数据库 ID
	Status                string         `json:"status" gorm:"not null" binding:"required,oneof=闲置 在用 待注销 已注销 待核实-办卡人离职"`   // 号码状态，添加 binding 和 oneof
	Vendor                string         `json:"vendor" binding:"max=100"`                                                  // 供应商
	Remarks               string         `json:"remarks" binding:"max=255"`                                                 // 备注
	CancellationDate      *time.Time     `json:"cancellationDate" binding:"omitempty,time_format=2006-01-02"`               // 注销日期
	CreatedAt             time.Time      `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt             time.Time      `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt             gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

// TableName 指定 MobileNumber 结构体对应的数据库表名
func (MobileNumber) TableName() string {
	return "mobile_numbers"
}

// MobileNumberResponse 是用于 API 响应的手机号码数据结构，包含关联信息
type MobileNumberResponse struct {
	ID                    uint                 `json:"id"`
	PhoneNumber           string               `json:"phoneNumber"`
	ApplicantEmployeeDbID uint                 `json:"applicantEmployeeId"`
	ApplicantName         string               `json:"applicantName,omitempty"`   // 办卡人姓名
	ApplicantStatus       string               `json:"applicantStatus,omitempty"` // 办卡人当前在职状态
	ApplicationDate       time.Time            `json:"applicationDate"`
	CurrentEmployeeDbID   *uint                `json:"currentEmployeeDbId,omitempty"`
	CurrentUserName       string               `json:"currentUserName,omitempty"` // 当前使用人姓名
	Status                string               `json:"status"`
	Vendor                string               `json:"vendor,omitempty"`
	Remarks               string               `json:"remarks,omitempty"`
	CancellationDate      *time.Time           `json:"cancellationDate,omitempty"`
	CreatedAt             time.Time            `json:"createdAt"`
	UpdatedAt             time.Time            `json:"updatedAt"`
	UsageHistory          []NumberUsageHistory `json:"usageHistory,omitempty"` // 号码使用历史
}

// MobileNumberUpdatePayload 定义了更新手机号码信息的请求体结构
type MobileNumberUpdatePayload struct {
	Status  *string `json:"status,omitempty" binding:"omitempty,oneof=闲置 在用 待注销 已注销 待核实-办卡人离职"` // 号码状态
	Vendor  *string `json:"vendor,omitempty" binding:"omitempty,max=100"`                       // 供应商
	Remarks *string `json:"remarks,omitempty" binding:"omitempty,max=255"`                      // 备注
}

// MobileNumberAssignPayload 定义了分配号码的请求体
// ... existing code ...
