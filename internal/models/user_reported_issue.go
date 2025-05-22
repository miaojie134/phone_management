package models

import (
	"database/sql"
	"time"
)

// UserReportedIssue represents an issue reported by a user.
type UserReportedIssue struct {
	ID                     uint           `gorm:"primaryKey"`
	VerificationTokenId    sql.NullInt64  // Nullable Foreign Key to VerificationTokens
	ReportedByEmployeeDbId uint           // Foreign Key to Employees
	MobileNumberDbId       sql.NullInt64  // Nullable Foreign Key to MobileNumbers
	ReportedPhoneNumber    sql.NullString // For unlisted numbers
	IssueType              string         // e.g., 'not_my_number', 'unlisted_number_in_use'
	UserComment            sql.NullString
	AdminActionStatus      string // e.g., 'pending_review', 'resolved'
	AdminRemarks           sql.NullString
	CreatedAt              time.Time
	UpdatedAt              time.Time
}
