package models

import (
	"time"
)

// NumberApplicantHistory 号码办卡人变更历史
type NumberApplicantHistory struct {
	ID                  uint      `json:"id" gorm:"primaryKey;autoIncrement"`                                       // 主键，自增
	MobileNumberDbID    uint      `json:"mobileNumberDbId" gorm:"column:mobile_number_db_id;not null"`              // 手机号码数据库ID
	PreviousApplicantID string    `json:"previousApplicantId" gorm:"column:previous_applicant_id;size:50;not null"` // 原办卡人员工业务工号
	NewApplicantID      string    `json:"newApplicantId" gorm:"column:new_applicant_id;size:50;not null"`           // 新办卡人员工业务工号
	ChangeDate          time.Time `json:"changeDate" gorm:"column:change_date;not null"`                            // 变更日期
	OperatorUsername    *string   `json:"operatorUsername" gorm:"column:operator_username;size:50"`                 // 系统操作员用户名
	Remarks             string    `json:"remarks" gorm:"column:remarks;size:500"`                                   // 备注
	CreatedAt           time.Time `json:"createdAt" gorm:"column:created_at;autoCreateTime"`                        // 记录创建时间
	UpdatedAt           time.Time `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`                        // 记录更新时间
}

// TableName 设置表名
func (NumberApplicantHistory) TableName() string {
	return "number_applicant_histories"
}
