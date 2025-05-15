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
