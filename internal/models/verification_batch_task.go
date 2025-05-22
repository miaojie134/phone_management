package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
