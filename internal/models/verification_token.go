package models

import (
	"time"
)

// VerificationToken represents the verification_tokens table
type VerificationToken struct {
	ID           uint      `gorm:"primaryKey;autoIncrement;not null"`
	EmployeeDbId uint      `gorm:"not null"`
	Token        string    `gorm:"type:varchar(255);unique;not null;index"`
	Status       string    `gorm:"type:varchar(50);not null;default:'pending'"`
	ExpiresAt    time.Time `gorm:"not null"`
	CreatedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}

// TableName specifies the table name for the VerificationToken model
func (VerificationToken) TableName() string {
	return "verification_tokens"
}
