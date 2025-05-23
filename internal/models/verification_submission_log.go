package models

import (
	"time"

	"gorm.io/gorm"
)

// VerificationActionType 定义了验证动作的类型
type VerificationActionType string

const (
	ActionConfirmUsage   VerificationActionType = "confirm_usage"
	ActionReportIssue    VerificationActionType = "report_issue"
	ActionReportUnlisted VerificationActionType = "report_unlisted"
)

// VerificationSubmissionLog 表示号码验证提交的日志记录
type VerificationSubmissionLog struct {
	ID                  uint                   `gorm:"primaryKey;autoIncrement;not null"`
	EmployeeID          string                 `gorm:"column:employee_id;not null;size:10;index"`
	VerificationTokenID uint                   `gorm:"column:verification_token_id;index"`
	MobileNumberID      *uint                  `gorm:"column:mobile_number_id;index"`
	PhoneNumber         string                 `gorm:"column:phone_number;size:20;index"`
	ActionType          VerificationActionType `gorm:"column:action_type;type:varchar(50);not null;index"`
	Purpose             *string                `gorm:"column:purpose;type:varchar(255)"`
	UserComment         *string                `gorm:"column:user_comment;type:text"`
	CreatedAt           time.Time              `gorm:"column:created_at;not null;autoCreateTime;index"`
	UpdatedAt           time.Time              `gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt           gorm.DeletedAt         `gorm:"index"`
}

// TableName 指定 VerificationSubmissionLog 模型对应的数据库表名
func (VerificationSubmissionLog) TableName() string {
	return "verification_submissions_log"
}

// PhoneVerificationStatusResponse 表示以手机号码维度统计的管理员视图响应结构
type PhoneVerificationStatusResponse struct {
	Summary         PhoneVerificationSummary     `json:"summary"`                   // 统计摘要
	ConfirmedPhones []ConfirmedPhoneDetail       `json:"confirmedPhones,omitempty"` // 已确认使用的手机号列表
	PendingUsers    []PendingUserDetail          `json:"pendingUsers,omitempty"`    // 未响应用户列表
	ReportedIssues  []ReportedIssueDetail        `json:"reportedIssues,omitempty"`  // 用户报告问题列表
	UnlistedNumbers []ReportedUnlistedNumberInfo `json:"unlistedNumbers,omitempty"` // 用户报告的未列出号码列表
}

// PhoneVerificationSummary 表示以手机号码维度统计的摘要
type PhoneVerificationSummary struct {
	TotalPhonesCount         int `json:"totalPhonesCount"`         // 系统中可用手机号码总数量（排除已注销）
	ConfirmedPhonesCount     int `json:"confirmedPhonesCount"`     // 已确认使用的手机号码数
	ReportedIssuesCount      int `json:"reportedIssuesCount"`      // 有问题的手机号码数
	PendingPhonesCount       int `json:"pendingPhonesCount"`       // 待确认的手机号码数
	NewlyReportedPhonesCount int `json:"newlyReportedPhonesCount"` // 新上报的手机号码数
}

// ConfirmedPhoneDetail 表示已确认使用的手机号码详情
type ConfirmedPhoneDetail struct {
	ID          uint      `json:"id"`                   // 手机号ID
	PhoneNumber string    `json:"phoneNumber"`          // 手机号码
	Department  string    `json:"department,omitempty"` // 部门
	CurrentUser string    `json:"currentUser"`          // 当前使用人
	Purpose     *string   `json:"purpose,omitempty"`    // 用途
	ConfirmedBy string    `json:"confirmedBy"`          // 确认人
	ConfirmedAt time.Time `json:"confirmedAt"`          // 确认时间
}
