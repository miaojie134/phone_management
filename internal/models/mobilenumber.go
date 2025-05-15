package models

import (
	"time"

	"gorm.io/gorm"
)

// MobileNumber 对应于数据库中的 mobile_numbers 表
type MobileNumber struct {
	ID                    int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	PhoneNumber           string         `json:"phoneNumber" gorm:"column:phone_number;unique;not null;size:50"`
	ApplicantEmployeeDbID int64          `json:"applicantEmployeeDbId" gorm:"column:applicant_employee_db_id;not null"` // 办卡人员工记录的数据库 ID
	ApplicationDate       time.Time      `json:"applicationDate" gorm:"column:application_date;type:date;not null"`     // 办卡日期
	CurrentEmployeeDbID   *int64         `json:"currentEmployeeDbId,omitempty" gorm:"column:current_employee_db_id"`    // 当前使用人员工记录的数据库 ID
	Status                string         `json:"status" gorm:"column:status;not null;default:'闲置';size:50"`             // 号码状态
	Vendor                *string        `json:"vendor,omitempty" gorm:"column:vendor;size:100"`                        // 供应商
	Remarks               *string        `json:"remarks,omitempty" gorm:"column:remarks;type:text"`                     // 备注
	CancellationDate      *time.Time     `json:"cancellationDate,omitempty" gorm:"column:cancellation_date;type:date"`  // 注销日期
	CreatedAt             time.Time      `json:"createdAt" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt             time.Time      `json:"updatedAt" gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt             gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

// TableName 指定 MobileNumber 结构体对应的数据库表名
func (MobileNumber) TableName() string {
	return "mobile_numbers"
}
