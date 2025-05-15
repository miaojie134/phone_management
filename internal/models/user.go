package models

import (
	"time"

	"gorm.io/gorm"
)

// User 对应于数据库中的 users 表
type User struct {
	ID           int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	Username     string         `json:"username" gorm:"column:username;unique;not null;size:255"`
	PasswordHash string         `json:"-" gorm:"column:password_hash;not null;size:255"` // 密码哈希不通过JSON暴露
	Role         string         `json:"role" gorm:"column:role;not null;default:'admin';size:50"`
	CreatedAt    time.Time      `json:"createdAt" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt    time.Time      `json:"updatedAt" gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

// TableName 指定 User 结构体对应的数据库表名
func (User) TableName() string {
	return "users"
}
