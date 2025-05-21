package models

import (
	"time"
)

// UserReportedIssue represents the user_reported_issues table
type UserReportedIssue struct {
	ID                     uint       `gorm:"primaryKey;autoIncrement;not null"`
	VerificationTokenId    *uint      `gorm:"null"` // Pointer to allow NULL values
	ReportedByEmployeeDbId uint       `gorm:"not null"`
	MobileNumberDbId       *uint      `gorm:"null"` // Pointer to allow NULL values
	ReportedPhoneNumber    *string    `gorm:"type:varchar(50);null"` // Pointer to allow NULL values
	IssueType              string     `gorm:"type:varchar(50);not null"`
	UserComment            *string    `gorm:"type:text;null"` // Pointer to allow NULL values
	AdminActionStatus      string     `gorm:"type:varchar(50);not null;default:'pending_review'"`
	AdminRemarks           *string    `gorm:"type:text;null"` // Pointer to allow NULL values
	CreatedAt              time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt              time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

// TableName specifies the table name for the UserReportedIssue model
func (UserReportedIssue) TableName() string {
	return "user_reported_issues"
}
