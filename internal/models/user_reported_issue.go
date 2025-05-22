package models

import (
	"time"

	"gorm.io/gorm"
)

// UserReportedIssue represents the user_reported_issues table
type UserReportedIssue struct {
	ID                   uint           `gorm:"primaryKey;autoIncrement;not null"`
	VerificationTokenId  *uint          `gorm:"null"` // Pointer to allow NULL values
	ReportedByEmployeeID string         `gorm:"column:reported_by_employee_id;not null;size:10"`
	MobileNumberDbId     *uint          `gorm:"null"`                  // Pointer to allow NULL values
	ReportedPhoneNumber  *string        `gorm:"type:varchar(50);null"` // Pointer to allow NULL values
	IssueType            string         `gorm:"type:varchar(50);not null"`
	UserComment          *string        `gorm:"type:text;null"` // Pointer to allow NULL values
	AdminActionStatus    string         `gorm:"type:varchar(50);not null;default:'pending_review'"`
	AdminRemarks         *string        `gorm:"type:text;null"` // Pointer to allow NULL values
	CreatedAt            time.Time      `json:"createdAt" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt            time.Time      `json:"updatedAt" gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt            gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index" swaggertype:"string" format:"date-time"`
}

// TableName specifies the table name for the UserReportedIssue model
func (UserReportedIssue) TableName() string {
	return "user_reported_issues"
}
