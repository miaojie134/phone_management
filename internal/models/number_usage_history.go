package models

import (
	"time"

	"gorm.io/gorm"
)

// NumberUsageHistory 对应于数据库中的 number_usage_history 表
type NumberUsageHistory struct {
	ID               int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	MobileNumberDbID int64          `json:"mobileNumberDbId" gorm:"column:mobile_number_db_id;not null"` // 手机号码记录的数据库 ID
	EmployeeID       string         `json:"employeeId" gorm:"column:employee_id;not null"`               // 使用人员工业务工号
	StartDate        time.Time      `json:"startDate" gorm:"column:start_date;not null"`                 // 使用开始日期时间
	EndDate          *time.Time     `json:"endDate,omitempty" gorm:"column:end_date"`                    // 使用结束日期时间
	CreatedAt        time.Time      `json:"createdAt" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt        time.Time      `json:"updatedAt" gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index" swaggertype:"string" format:"date-time"`
}

// TableName 指定 NumberUsageHistory 结构体对应的数据库表名
func (NumberUsageHistory) TableName() string {
	return "number_usage_history"
}
