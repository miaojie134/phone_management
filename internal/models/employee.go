package models

import (
	"time"

	"gorm.io/gorm"
)

// Employee 对应于数据库中的 employees 表
type Employee struct {
	ID               int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	EmployeeID       string         `json:"employeeId" gorm:"column:employee_id;unique;not null;size:100"`                      // 员工业务工号
	FullName         string         `json:"fullName" gorm:"column:full_name;not null;size:255"`                                 // 姓名
	Department       *string        `json:"department,omitempty" gorm:"column:department;size:255"`                             // 部门
	EmploymentStatus string         `json:"employmentStatus" gorm:"column:employment_status;not null;default:'Active';size:50"` // 在职状态 (例如: 'Active', 'Departed')
	HireDate         *time.Time     `json:"hireDate,omitempty" gorm:"column:hire_date;type:date"`                               // 入职日期
	TerminationDate  *time.Time     `json:"terminationDate,omitempty" gorm:"column:termination_date;type:date"`                 // 离职日期
	CreatedAt        time.Time      `json:"createdAt" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt        time.Time      `json:"updatedAt" gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

// TableName 指定 Employee 结构体对应的数据库表名
func (Employee) TableName() string {
	return "employees"
}

// EmployeeDetailResponse 定义了获取员工详情时的响应结构
// 包含员工基本信息及其作为"办卡人"和"当前使用人"的号码简要列表
type EmployeeDetailResponse struct {
	ID                   int64                   `json:"id"`
	EmployeeID           string                  `json:"employeeId"`
	FullName             string                  `json:"fullName"`
	Department           *string                 `json:"department,omitempty"`
	EmploymentStatus     string                  `json:"employmentStatus"`
	HireDate             *time.Time              `json:"hireDate,omitempty"`
	TerminationDate      *time.Time              `json:"terminationDate,omitempty"`
	CreatedAt            time.Time               `json:"createdAt"`
	UpdatedAt            time.Time               `json:"updatedAt"`
	HandledMobileNumbers []MobileNumberBasicInfo `json:"handledMobileNumbers,omitempty"` // 作为办卡人的号码列表
	UsingMobileNumbers   []MobileNumberBasicInfo `json:"usingMobileNumbers,omitempty"`   // 作为当前使用人的号码列表
}
