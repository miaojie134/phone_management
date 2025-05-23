package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// =========== 验证令牌相关 ===========

// VerificationScopeType 定义了验证流程的范围类型
type VerificationScopeType string

// VerificationTokenStatus 定义了验证令牌的状态类型
type VerificationTokenStatus string

const (
	VerificationScopeAllUsers    VerificationScopeType = "all_users"
	VerificationScopeDepartment  VerificationScopeType = "department"
	VerificationScopeEmployeeIDs VerificationScopeType = "employee_ids"

	VerificationTokenStatusPending VerificationTokenStatus = "pending"
	VerificationTokenStatusExpired VerificationTokenStatus = "expired"
)

// VerificationToken represents the verification_tokens table
type VerificationToken struct {
	ID         uint                    `gorm:"primaryKey;autoIncrement;not null"`
	EmployeeID string                  `gorm:"column:employee_id;not null;size:10"`
	Token      string                  `gorm:"type:varchar(255);unique;not null;index"`
	Status     VerificationTokenStatus `gorm:"type:varchar(50);not null;default:'pending'"`
	ExpiresAt  time.Time               `gorm:"not null"`
	CreatedAt  time.Time               `json:"createdAt" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt  time.Time               `json:"updatedAt" gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt  gorm.DeletedAt          `json:"deletedAt,omitempty" gorm:"index" swaggertype:"string" format:"date-time"`
}

// TableName specifies the table name for the VerificationToken model
func (VerificationToken) TableName() string {
	return "verification_tokens"
}

// =========== 批量验证任务相关 ===========

// VerificationBatchTaskStatus 定义了批量验证任务的状态类型
type VerificationBatchTaskStatus string

const (
	BatchTaskStatusPending             VerificationBatchTaskStatus = "Pending"             // 任务已创建，等待处理
	BatchTaskStatusInProgress          VerificationBatchTaskStatus = "InProgress"          // 任务正在处理中
	BatchTaskStatusCompleted           VerificationBatchTaskStatus = "Completed"           // 任务成功完成
	BatchTaskStatusCompletedWithErrors VerificationBatchTaskStatus = "CompletedWithErrors" // 任务完成，但有部分失败
	BatchTaskStatusFailed              VerificationBatchTaskStatus = "Failed"              // 任务未能成功发起或执行（例如，初始查找员工就失败）
)

// VerificationBatchTask 代表一个号码验证的批处理任务
type VerificationBatchTask struct {
	ID                      string                      `json:"id" gorm:"type:varchar(36);primaryKey"` // 使用 UUID 作为主键
	Status                  VerificationBatchTaskStatus `json:"status" gorm:"type:varchar(50);not null;index"`
	TotalEmployeesToProcess int                         `json:"totalEmployeesToProcess" gorm:"not null"`
	TokensGeneratedCount    int                         `json:"tokensGeneratedCount" gorm:"not null;default:0"`
	EmailsAttemptedCount    int                         `json:"emailsAttemptedCount" gorm:"not null;default:0"`
	EmailsSucceededCount    int                         `json:"emailsSucceededCount" gorm:"not null;default:0"`
	EmailsFailedCount       int                         `json:"emailsFailedCount" gorm:"not null;default:0"`
	ErrorSummary            *string                     `json:"errorSummary,omitempty" gorm:"type:text"` // 存储JSON数组字符串或其他格式的错误概要
	RequestedScopeType      VerificationScopeType       `json:"requestedScopeType" gorm:"type:varchar(50)"`
	RequestedScopeValues    *string                     `json:"requestedScopeValues,omitempty" gorm:"type:text"` // 例如 JSON 数组字符串
	RequestedDurationDays   int                         `json:"requestedDurationDays"`
	CreatedAt               time.Time                   `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt               time.Time                   `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt               gorm.DeletedAt              `json:"deletedAt,omitempty" gorm:"index"`
}

// TableName 指定 VerificationBatchTask 模型对应的数据库表名
func (VerificationBatchTask) TableName() string {
	return "verification_batch_tasks"
}

// BeforeCreate GORM hook 为 VerificationBatchTask 生成 UUID
func (task *VerificationBatchTask) BeforeCreate(tx *gorm.DB) (err error) {
	if task.ID == "" {
		task.ID = uuid.NewString()
	}
	return nil
}

// EmailFailureDetail 用于在 ErrorSummary 中记录单个邮件发送失败的详情
// 可以考虑将 ErrorSummary 存储为 []EmailFailureDetail 的 JSON 字符串
type EmailFailureDetail struct {
	EmployeeID   string `json:"employeeId"`
	EmployeeName string `json:"employeeName"`
	EmailAddress string `json:"emailAddress"`
	Reason       string `json:"reason"`
}

// =========== 验证结果提交相关 ===========

// VerifiedNumber 表示用户确认的号码信息
type VerifiedNumber struct {
	MobileNumberId uint    `json:"mobileNumberId" binding:"required"`
	Action         string  `json:"action" binding:"required,oneof=confirm_usage report_issue"` // confirm_usage, report_issue
	Purpose        *string `json:"purpose,omitempty"`                                          // 用户确认或报告问题时可提供的号码用途
	UserComment    string  `json:"userComment,omitempty"`
}

// UnlistedNumber 表示用户报告的未在系统中列出的号码
type UnlistedNumber struct {
	PhoneNumber string  `json:"phoneNumber" binding:"required,len=11,numeric"`
	Purpose     *string `json:"purpose" binding:"required,max=255"` // 用户报告该号码的用途
	UserComment string  `json:"userComment,omitempty"`
}

// VerificationSubmission 表示用户提交的号码确认结果
type VerificationSubmission struct {
	VerifiedNumbers         []VerifiedNumber `json:"verifiedNumbers" binding:"omitempty,dive"`
	UnlistedNumbersReported []UnlistedNumber `json:"unlistedNumbersReported,omitempty" binding:"omitempty,dive"`
}

// VerificationPhoneNumber 表示验证流程中的手机号码信息
type VerificationPhoneNumber struct {
	ID          uint    `json:"id"`
	PhoneNumber string  `json:"phoneNumber"`
	Department  string  `json:"department"`
	Purpose     *string `json:"purpose,omitempty"`     // 号码用途
	Status      string  `json:"status"`                // pending, confirmed, reported
	UserComment *string `json:"userComment,omitempty"` // 用户报告问题时的评论
}

// VerificationInfo 表示验证信息的响应结构
type VerificationInfo struct {
	EmployeeID                 string                       `json:"employeeId"`
	EmployeeName               string                       `json:"employeeName"`
	PhoneNumbers               []VerificationPhoneNumber    `json:"phoneNumbers"`
	PreviouslyReportedUnlisted []ReportedUnlistedNumberInfo `json:"previouslyReportedUnlisted,omitempty"`
	ExpiresAt                  time.Time                    `json:"expiresAt"`
}

// ReportedUnlistedNumberInfo 表示用户已报告的未列出号码的信息
type ReportedUnlistedNumberInfo struct {
	PhoneNumber string    `json:"phoneNumber"`
	UserComment string    `json:"userComment,omitempty"`
	Purpose     *string   `json:"purpose,omitempty"` // 新增字段：用户报告该未列出号码时的用途
	ReportedAt  time.Time `json:"reportedAt"`
}

// PendingUserDetail 表示未响应确认的用户详情
type PendingUserDetail struct {
	EmployeeID string     `json:"employeeId"`          // 员工业务工号
	FullName   string     `json:"fullName"`            // 员工姓名
	Email      *string    `json:"email,omitempty"`     // 员工邮箱
	TokenID    uint       `json:"tokenId,omitempty"`   // 令牌ID（可选，用于内部处理）
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"` // 令牌过期时间（可选）
}

// ReportedIssueDetail 表示用户报告的号码问题详情
type ReportedIssueDetail struct {
	IssueID           uint      `json:"issueId,omitempty"`           // 问题ID（可选，用于内部处理）
	PhoneNumber       string    `json:"phoneNumber"`                 // 手机号码
	ReportedBy        string    `json:"reportedBy"`                  // 报告人姓名
	Comment           string    `json:"comment"`                     // 用户备注
	Purpose           *string   `json:"purpose,omitempty"`           // 报告的用途
	OriginalStatus    string    `json:"originalStatus"`              // 号码原始状态
	ReportedAt        time.Time `json:"reportedAt"`                  // 报告时间
	AdminActionStatus string    `json:"adminActionStatus,omitempty"` // 管理员处理状态
}
