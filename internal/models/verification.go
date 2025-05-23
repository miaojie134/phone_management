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
	BatchTaskStatusPending             VerificationBatchTaskStatus = "Pending"
	BatchTaskStatusInProgress          VerificationBatchTaskStatus = "InProgress"
	BatchTaskStatusCompleted           VerificationBatchTaskStatus = "Completed"
	BatchTaskStatusCompletedWithErrors VerificationBatchTaskStatus = "CompletedWithErrors"
	BatchTaskStatusFailed              VerificationBatchTaskStatus = "Failed"
)

// VerificationBatchTask 代表一个号码验证的批处理任务
type VerificationBatchTask struct {
	ID                      string                      `json:"id" gorm:"type:varchar(36);primaryKey"`
	Status                  VerificationBatchTaskStatus `json:"status" gorm:"type:varchar(50);not null;index"`
	TotalEmployeesToProcess int                         `json:"totalEmployeesToProcess" gorm:"not null"`
	TokensGeneratedCount    int                         `json:"tokensGeneratedCount" gorm:"not null;default:0"`
	EmailsAttemptedCount    int                         `json:"emailsAttemptedCount" gorm:"not null;default:0"`
	EmailsSucceededCount    int                         `json:"emailsSucceededCount" gorm:"not null;default:0"`
	EmailsFailedCount       int                         `json:"emailsFailedCount" gorm:"not null;default:0"`
	ErrorSummary            *string                     `json:"errorSummary,omitempty" gorm:"type:text"`
	RequestedScopeType      VerificationScopeType       `json:"requestedScopeType" gorm:"type:varchar(50)"`
	RequestedScopeValues    *string                     `json:"requestedScopeValues,omitempty" gorm:"type:text"`
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
type EmailFailureDetail struct {
	EmployeeID   string `json:"employeeId"`
	EmployeeName string `json:"employeeName"`
	EmailAddress string `json:"emailAddress"`
	Reason       string `json:"reason"`
}

// =========== 验证结果提交相关 (API DTOs) ===========

// VerifiedNumber 表示用户确认的号码信息
type VerifiedNumber struct {
	MobileNumberId uint    `json:"mobileNumberId" binding:"required"`
	Action         string  `json:"action" binding:"required,oneof=confirm_usage report_issue"`
	Purpose        *string `json:"purpose,omitempty"`
	UserComment    string  `json:"userComment,omitempty"`
}

// UnlistedNumber 表示用户报告的未在系统中列出的号码
type UnlistedNumber struct {
	PhoneNumber string  `json:"phoneNumber" binding:"required,len=11,numeric"`
	Purpose     *string `json:"purpose" binding:"required,max=255"`
	UserComment string  `json:"userComment,omitempty"`
}

// VerificationSubmission 表示用户提交的号码确认结果
type VerificationSubmission struct {
	VerifiedNumbers         []VerifiedNumber `json:"verifiedNumbers" binding:"omitempty,dive"`
	UnlistedNumbersReported []UnlistedNumber `json:"unlistedNumbersReported,omitempty" binding:"omitempty,dive"`
}

// =========== 用户获取验证信息 (API DTOs) ===========

// VerificationPhoneNumber 表示验证流程中的手机号码信息
type VerificationPhoneNumber struct {
	ID          uint    `json:"id"`
	PhoneNumber string  `json:"phoneNumber"`
	Department  string  `json:"department"`
	Purpose     *string `json:"purpose,omitempty"`
	Status      string  `json:"status"`
	UserComment *string `json:"userComment,omitempty"`
}

// VerificationInfo 表示验证信息的响应结构
type VerificationInfo struct {
	EmployeeID                 string                       `json:"employeeId"`
	EmployeeName               string                       `json:"employeeName"`
	PhoneNumbers               []VerificationPhoneNumber    `json:"phoneNumbers"`
	PreviouslyReportedUnlisted []ReportedUnlistedNumberInfo `json:"previouslyReportedUnlisted,omitempty"`
	ExpiresAt                  time.Time                    `json:"expiresAt"`
}

// ReportedUnlistedNumberInfo 表示用户已报告的未列出号码的信息 (用于 VerificationInfo)
type ReportedUnlistedNumberInfo struct {
	PhoneNumber string    `json:"phoneNumber"`
	UserComment string    `json:"userComment,omitempty"`
	Purpose     *string   `json:"purpose,omitempty"`
	ReportedAt  time.Time `json:"reportedAt"`
}

// =========== 管理员视图和日志相关 (API DTOs & Log Model) ===========

// PendingUserDetail 表示未响应确认的用户详情 (用于 PhoneVerificationStatusResponse)
type PendingUserDetail struct {
	EmployeeID string     `json:"employeeId"`
	FullName   string     `json:"fullName"`
	Email      *string    `json:"email,omitempty"`
	TokenID    uint       `json:"tokenId,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
}

// ReportedIssueDetail 表示用户报告的号码问题详情 (用于 PhoneVerificationStatusResponse)
type ReportedIssueDetail struct {
	IssueID           uint      `json:"issueId,omitempty"`
	PhoneNumber       string    `json:"phoneNumber"`
	ReportedBy        string    `json:"reportedBy"`
	Comment           string    `json:"comment"`
	Purpose           *string   `json:"purpose,omitempty"`
	OriginalStatus    string    `json:"originalStatus"`
	ReportedAt        time.Time `json:"reportedAt"`
	AdminActionStatus string    `json:"adminActionStatus,omitempty"`
}

// VerificationActionType 定义了验证动作的类型
type VerificationActionType string

const (
	ActionConfirmUsage   VerificationActionType = "confirm_usage"
	ActionReportIssue    VerificationActionType = "report_issue"
	ActionReportUnlisted VerificationActionType = "report_unlisted"
)

// VerificationSubmissionLog 表示号码验证提交的日志记录 (数据库表模型)
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

// PhoneVerificationStatusResponse 表示以手机号码维度统计的管理员视图响应结构 (API DTO)
type PhoneVerificationStatusResponse struct {
	Summary         PhoneVerificationSummary     `json:"summary"`
	ConfirmedPhones []ConfirmedPhoneDetail       `json:"confirmedPhones,omitempty"`
	PendingUsers    []PendingUserDetail          `json:"pendingUsers,omitempty"`    // Reuses PendingUserDetail
	ReportedIssues  []ReportedIssueDetail        `json:"reportedIssues,omitempty"`  // Reuses ReportedIssueDetail
	UnlistedNumbers []ReportedUnlistedNumberInfo `json:"unlistedNumbers,omitempty"` // Reuses ReportedUnlistedNumberInfo
}

// PhoneVerificationSummary 表示以手机号码维度统计的摘要 (用于 PhoneVerificationStatusResponse)
type PhoneVerificationSummary struct {
	TotalPhonesCount         int `json:"totalPhonesCount"`
	ConfirmedPhonesCount     int `json:"confirmedPhonesCount"`
	ReportedIssuesCount      int `json:"reportedIssuesCount"`
	PendingPhonesCount       int `json:"pendingPhonesCount"`
	NewlyReportedPhonesCount int `json:"newlyReportedPhonesCount"`
}

// ConfirmedPhoneDetail 表示已确认使用的手机号码详情 (用于 PhoneVerificationStatusResponse)
type ConfirmedPhoneDetail struct {
	ID          uint      `json:"id"`
	PhoneNumber string    `json:"phoneNumber"`
	Department  string    `json:"department,omitempty"`
	CurrentUser string    `json:"currentUser"`
	Purpose     *string   `json:"purpose,omitempty"`
	ConfirmedBy string    `json:"confirmedBy"`
	ConfirmedAt time.Time `json:"confirmedAt"`
}
