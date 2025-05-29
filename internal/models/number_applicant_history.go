package models

import (
	"time"

	"gorm.io/gorm"
)

// NumberApplicantHistory 号码办卡人变更历史表
type NumberApplicantHistory struct {
	ID                  uint           `json:"id" gorm:"primaryKey"`
	MobileNumberDbID    uint           `json:"mobileNumberDbId" gorm:"column:mobile_number_db_id;not null"`      // 关联MobileNumbers.id
	PreviousApplicantID string         `json:"previousApplicantId" gorm:"column:previous_applicant_id;not null"` // 原办卡人员工业务工号
	NewApplicantID      string         `json:"newApplicantId" gorm:"column:new_applicant_id;not null"`           // 新办卡人员工业务工号
	ChangeDate          time.Time      `json:"changeDate" gorm:"column:change_date;not null"`                    // 变更日期
	ChangeReason        string         `json:"changeReason" gorm:"column:change_reason;size:255"`                // 变更原因
	OperatorEmployeeID  *string        `json:"operatorEmployeeId,omitempty" gorm:"column:operator_employee_id"`  // 操作人员工业务工号
	Remarks             string         `json:"remarks" gorm:"size:500"`                                          // 备注
	CreatedAt           time.Time      `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt           time.Time      `json:"updatedAt" gorm:"autoUpdateTime"`
	DeletedAt           gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

// TableName 指定 NumberApplicantHistory 结构体对应的数据库表名
func (NumberApplicantHistory) TableName() string {
	return "number_applicant_histories"
}
