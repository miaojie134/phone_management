package models

import (
	"time"

	"gorm.io/gorm"
)

// VerificationToken represents the verification_tokens table
type VerificationToken struct {
	ID         uint           `gorm:"primaryKey;autoIncrement;not null"`
	EmployeeID string         `gorm:"column:employee_id;not null;size:10"`
	Token      string         `gorm:"type:varchar(255);unique;not null;index"`
	Status     string         `gorm:"type:varchar(50);not null;default:'pending'"`
	ExpiresAt  time.Time      `gorm:"not null"`
	CreatedAt  time.Time      `json:"createdAt" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt  time.Time      `json:"updatedAt" gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt  gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index" swaggertype:"string" format:"date-time"`
}

// TableName specifies the table name for the VerificationToken model
func (VerificationToken) TableName() string {
	return "verification_tokens"
}
