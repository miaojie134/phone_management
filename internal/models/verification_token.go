package models

import "time"

// VerificationToken represents a token used for verifying employee actions.
type VerificationToken struct {
	ID           uint      `gorm:"primaryKey"`
	EmployeeDbId uint      // Foreign Key to Employees
	Token        string    `gorm:"uniqueIndex"`
	Status       string    // e.g., 'pending', 'used', 'expired'
	ExpiresAt    time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
