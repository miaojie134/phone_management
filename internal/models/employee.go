package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Employee 对应于数据库中的 employees 表
type Employee struct {
	ID               int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	EmployeeID       string         `json:"employeeId" gorm:"column:employee_id;unique;not null;size:10"`                                      // 员工业务工号, 例如 EMP0000001
	FullName         string         `json:"fullName" gorm:"column:full_name;not null;size:255"`                                                // 姓名
	PhoneNumber      *string        `json:"phoneNumber,omitempty" gorm:"column:phone_number;size:11;uniqueIndex:idx_phone_number_not_deleted"` // 员工手机号码, 11位, 可选, 唯一 (NULLS NOT DISTINCT)
	Email            *string        `json:"email,omitempty" gorm:"column:email;size:255;uniqueIndex:idx_email_not_deleted"`                    // 员工邮箱, 可选, 唯一 (NULLS NOT DISTINCT)
	Department       *string        `json:"department,omitempty" gorm:"column:department;size:255"`                                            // 部门
	EmploymentStatus string         `json:"employmentStatus" gorm:"column:employment_status;not null;default:'Active';size:50"`                // 在职状态 (例如: 'Active', 'Departed')
	HireDate         *time.Time     `json:"hireDate,omitempty" gorm:"column:hire_date;type:date"`                                              // 入职日期
	TerminationDate  *time.Time     `json:"terminationDate,omitempty" gorm:"column:termination_date;type:date"`                                // 离职日期
	CreatedAt        time.Time      `json:"createdAt" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt        time.Time      `json:"updatedAt" gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index" swaggertype:"string" format:"date-time"`
}

// TableName 指定 Employee 结构体对应的数据库表名
func (Employee) TableName() string {
	return "employees"
}

// BeforeCreate GORM Hook: 在创建员工记录前设置一个临时的 EmployeeID
func (e *Employee) BeforeCreate(tx *gorm.DB) (err error) {
	// 仅当 EmployeeID 为空时才设置临时ID，以允许在某些场景下可能预设ID（尽管当前流程不会）
	if e.EmployeeID == "" {
		// 使用 "TEMP_" 前缀加上纳秒时间戳作为临时唯一ID，以通过可能的 NOT NULL 和 UNIQUE 约束
		// 这个临时ID将在 AfterCreate Hook 中被替换
		e.EmployeeID = "TEMP_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return nil
}

// AfterCreate GORM Hook: 在创建员工记录并获得自增ID后，生成最终的 EmployeeID
func (e *Employee) AfterCreate(tx *gorm.DB) (err error) {
	// 检查当前 EmployeeID 是否是之前设置的临时ID，或者是否因为某些原因仍为空
	if strings.HasPrefix(e.EmployeeID, "TEMP_") || e.EmployeeID == "" {
		// 根据数据库自增的 e.ID 生成最终的 EmployeeID, 格式为 EMP0000001 (7位数字，总长10)
		newEmployeeID := fmt.Sprintf("EMP%07d", e.ID)

		// 更新数据库中该记录的 employee_id 字段
		// 这里必须使用 tx.Model(e).UpdateColumn(...) 或 tx.Model(&Employee{}).Where("id = ?", e.ID).UpdateColumn(...)
		// UpdateColumn 只更新指定字段，不会触发其他回调，且不会更新 updated_at (除非手动指定)
		// 如果用 Update, 它可能会触发其他hook，并且会更新 updated_at。
		// 为了精确控制，这里我们仅更新 employee_id
		err = tx.Model(e).UpdateColumn("employee_id", newEmployeeID).Error
		if err == nil {
			// 如果数据库更新成功，也更新内存中模型实例的 EmployeeID
			e.EmployeeID = newEmployeeID
		} else {
			// 如果更新失败，错误将由GORM处理（通常会回滚事务）
			return err
		}
	}
	return nil
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

// UpdateEmployeePayload 定义了更新员工请求的 JSON 结构体
// 所有字段都是可选的，因此使用指针类型
// 这个结构体用于API层的数据绑定和校验，并传递给服务层。
type UpdateEmployeePayload struct {
	Department       *string `json:"department,omitempty" binding:"omitempty,max=255"`
	EmploymentStatus *string `json:"employmentStatus,omitempty" binding:"omitempty,oneof=Active Inactive Departed"` // 校验允许的值
	TerminationDate  *string `json:"terminationDate,omitempty" binding:"omitempty,datetime=2006-01-02"`             // 日期格式 YYYY-MM-DD
}
