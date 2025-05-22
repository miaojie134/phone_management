package repositories

import (
	"context"
	"errors"
	"strings"

	"github.com/phone_management/internal/models"
	"gorm.io/gorm"
)

// ErrEmployeeIDExists 表示员工工号已存在
var ErrEmployeeIDExists = errors.New("员工工号已存在")

// ErrEmployeePhoneNumberConflict 表示员工的手机号码已存在
var ErrEmployeePhoneNumberConflict = errors.New("员工手机号码已存在")

// ErrEmployeeEmailConflict 表示员工的邮箱已存在
var ErrEmployeeEmailConflict = errors.New("员工邮箱已存在")

// EmployeeRepository 定义了员工数据仓库的接口
type EmployeeRepository interface {
	CreateEmployee(employee *models.Employee) (*models.Employee, error)
	GetEmployees(page, limit int, sortBy, sortOrder, search, employmentStatus string) ([]models.Employee, int64, error)
	GetEmployeeDetailByEmployeeID(employeeID string) (*models.EmployeeDetailResponse, error)
	GetEmployeeByEmployeeID(employeeID string) (*models.Employee, error)
	GetEmployeeByPhoneNumber(phoneNumber string) (*models.Employee, error)
	GetEmployeeByEmail(email string) (*models.Employee, error)
	UpdateEmployee(employeeID string, updates map[string]interface{}) (*models.Employee, error)
	GetEmployeesByFullName(fullName string) ([]*models.Employee, error)
	FindAllActive(ctx context.Context) ([]models.Employee, error)
	FindActiveByDepartmentNames(ctx context.Context, departmentNames []string) ([]models.Employee, error)
	FindActiveByEmployeeIDs(ctx context.Context, employeeIDs []string) ([]models.Employee, error)
	// 未来可以扩展其他方法，如 GetEmployeeByID, UpdateEmployee, DeleteEmployee 等
}

// gormEmployeeRepository 是 EmployeeRepository 的 GORM 实现
type gormEmployeeRepository struct {
	db *gorm.DB
}

// NewGormEmployeeRepository 创建一个新的 gormEmployeeRepository 实例
func NewGormEmployeeRepository(db *gorm.DB) EmployeeRepository {
	return &gormEmployeeRepository{db: db}
}

// CreateEmployee 在数据库中创建一个新的员工记录
func (r *gormEmployeeRepository) CreateEmployee(employee *models.Employee) (*models.Employee, error) {
	// EmployeeID 的生成由 model hooks (BeforeCreate, AfterCreate) 处理
	// 在 AfterCreate hook 中，会执行一次 update 来设置最终的 EmployeeID
	// 因此，这里的 Create 操作实际上是用一个临时的 EmployeeID (如果 BeforeCreate hook 被触发了)

	if err := r.db.Create(employee).Error; err != nil {
		// 检查是否是已知的唯一约束错误
		// 注意：错误字符串的判断可能因数据库类型而异，这里尝试覆盖常见情况
		lowerErr := strings.ToLower(err.Error())
		if strings.Contains(lowerErr, "unique constraint") || strings.Contains(lowerErr, "duplicate key") || strings.Contains(lowerErr, "duplicate entry") {
			if strings.Contains(lowerErr, "employee_id") || strings.Contains(lowerErr, "employees_employee_id_key") /* PostgreSQL specific key name example */ {
				return nil, ErrEmployeeIDExists
			}
			if strings.Contains(lowerErr, "phone_number") || strings.Contains(lowerErr, "idx_phone_number_not_deleted") {
				return nil, ErrEmployeePhoneNumberConflict
			}
			if strings.Contains(lowerErr, "email") || strings.Contains(lowerErr, "idx_email_not_deleted") {
				return nil, ErrEmployeeEmailConflict
			}
		}
		return nil, err // 返回原始错误，如果不是已知的唯一约束冲突或无法判断
	}
	// 创建成功后，employee 对象中的 EmployeeID 应该已经被 AfterCreate hook 更新
	return employee, nil
}

// GetEmployees 从数据库中获取员工列表，支持分页、排序、搜索和筛选
func (r *gormEmployeeRepository) GetEmployees(page, limit int, sortBy, sortOrder, search, employmentStatus string) ([]models.Employee, int64, error) {
	var employees []models.Employee
	var totalItems int64

	tx := r.db.Model(&models.Employee{})

	// 处理搜索条件 (匹配姓名、工号)
	if search != "" {
		searchTerm := "%" + search + "%"
		tx = tx.Where("full_name LIKE ? OR employee_id LIKE ?", searchTerm, searchTerm)
	}

	// 处理在职状态筛选
	if employmentStatus != "" {
		tx = tx.Where("employment_status = ?", employmentStatus)
	}

	// 计算总数（在应用分页之前）
	if err := tx.Count(&totalItems).Error; err != nil {
		return nil, 0, err
	}

	// 处理排序
	if sortBy != "" {
		// 白名单校验 sortBy 字段，防止 SQL 注入
		allowedSortByFields := map[string]string{
			"employeeId":       "employee_id",
			"fullName":         "full_name",
			"department":       "department",
			"employmentStatus": "employment_status",
			"hireDate":         "hire_date",
			"createdAt":        "created_at",
		}
		dbSortBy, isValidField := allowedSortByFields[sortBy]
		if !isValidField {
			dbSortBy = "created_at" // 如果字段无效，则使用默认排序字段
		}

		if strings.ToLower(sortOrder) != "desc" {
			sortOrder = "asc"
		}
		tx = tx.Order(dbSortBy + " " + sortOrder)
	} else {
		// 默认排序
		tx = tx.Order("created_at desc")
	}

	// 处理分页
	offset := (page - 1) * limit
	if err := tx.Offset(offset).Limit(limit).Find(&employees).Error; err != nil {
		return nil, 0, err
	}

	return employees, totalItems, nil
}

// GetEmployeeDetailByEmployeeID 从数据库中获取指定业务工号的员工详情及其关联的手机号码信息
func (r *gormEmployeeRepository) GetEmployeeDetailByEmployeeID(employeeID string) (*models.EmployeeDetailResponse, error) {
	var employee models.Employee
	// 通过业务工号 employee_id 查询员工
	if err := r.db.Where("employee_id = ?", employeeID).First(&employee).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound // 使用仓库层已定义的错误
		}
		return nil, err
	}

	empDetailResp := &models.EmployeeDetailResponse{
		ID:               employee.ID, // 仍然可以返回数据库ID
		EmployeeID:       employee.EmployeeID,
		FullName:         employee.FullName,
		Department:       employee.Department,
		EmploymentStatus: employee.EmploymentStatus,
		HireDate:         employee.HireDate,
		TerminationDate:  employee.TerminationDate,
		CreatedAt:        employee.CreatedAt,
		UpdatedAt:        employee.UpdatedAt,
	}

	// 获取作为办卡人的号码列表 (简要信息) - 查询条件已是 employee.EmployeeID (业务工号)
	var handledNumbers []models.MobileNumberBasicInfo
	if err := r.db.Model(&models.MobileNumber{}).
		Select("id, phone_number, status").
		Where("applicant_employee_id = ?", employee.EmployeeID).
		Find(&handledNumbers).Error; err != nil {
		return nil, err
	}
	empDetailResp.HandledMobileNumbers = handledNumbers

	// 获取作为当前使用人的号码列表 (简要信息) - 查询条件已是 employee.EmployeeID (业务工号)
	var usingNumbers []models.MobileNumberBasicInfo
	if err := r.db.Model(&models.MobileNumber{}).
		Select("id, phone_number, status").
		Where("current_employee_id = ?", employee.EmployeeID).
		Find(&usingNumbers).Error; err != nil {
		return nil, err
	}
	empDetailResp.UsingMobileNumbers = usingNumbers

	return empDetailResp, nil
}

// GetEmployeeByEmployeeID 根据员工业务工号 (employee_id 字段) 查询员工信息
func (r *gormEmployeeRepository) GetEmployeeByEmployeeID(employeeID string) (*models.Employee, error) {
	var employee models.Employee
	if err := r.db.Where("employee_id = ?", employeeID).First(&employee).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound // 复用已定义的记录未找到错误
		}
		return nil, err // 其他数据库错误
	}
	return &employee, nil
}

// GetEmployeeByPhoneNumber 根据手机号码查询员工 (排除软删除的)
func (r *gormEmployeeRepository) GetEmployeeByPhoneNumber(phoneNumber string) (*models.Employee, error) {
	var employee models.Employee
	if err := r.db.Where("phone_number = ?", phoneNumber).First(&employee).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound // 复用已定义的记录未找到错误
		}
		return nil, err
	}
	return &employee, nil
}

// GetEmployeeByEmail 根据邮箱查询员工 (排除软删除的)
func (r *gormEmployeeRepository) GetEmployeeByEmail(email string) (*models.Employee, error) {
	var employee models.Employee
	if err := r.db.Where("email = ?", email).First(&employee).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	return &employee, nil
}

// GetEmployeesByFullName 根据员工全名查找员工 (可能返回多个，用于处理重名场景)
func (r *gormEmployeeRepository) GetEmployeesByFullName(fullName string) ([]*models.Employee, error) {
	var employees []*models.Employee
	if err := r.db.Where("full_name = ?", fullName).Find(&employees).Error; err != nil {
		return nil, err
	}
	return employees, nil
}

// UpdateEmployee 更新指定业务工号的员工信息
func (r *gormEmployeeRepository) UpdateEmployee(employeeID string, updates map[string]interface{}) (*models.Employee, error) {
	// 首先，检查员工是否存在，确保我们操作的是一个有效的记录
	// GetEmployeeByEmployeeID 方法内部已经处理了 ErrRecordNotFound
	var employee models.Employee
	if err := r.db.Where("employee_id = ?", employeeID).First(&employee).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound // 如果员工不存在，返回错误
		}
		return nil, err // 其他数据库查询错误
	}

	// 更新记录
	// 使用 Model(&models.Employee{}) 指定模型，并通过 Where 更新特定 employee_id 的记录
	if err := r.db.Model(&models.Employee{}).Where("employee_id = ?", employeeID).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 重新查询更新后的记录并返回
	// 直接使用之前查询到的 employee 变量的指针，并让 GORM 通过其主键刷新
	// 或者更可靠地，再次通过 employeeID 查询
	var updatedEmployee models.Employee
	if err := r.db.Where("employee_id = ?", employeeID).First(&updatedEmployee).Error; err != nil {
		return nil, err // 理论上此时应该能找到
	}
	return &updatedEmployee, nil
}

// FindAllActive 查询所有在职员工
func (r *gormEmployeeRepository) FindAllActive(ctx context.Context) ([]models.Employee, error) {
	var employees []models.Employee
	if err := r.db.WithContext(ctx).Where("employment_status = ?", "Active").Find(&employees).Error; err != nil {
		return nil, err
	}
	return employees, nil
}

// FindActiveByDepartmentNames 查询指定部门名称列表中的所有在职员工
func (r *gormEmployeeRepository) FindActiveByDepartmentNames(ctx context.Context, departmentNames []string) ([]models.Employee, error) {
	var employees []models.Employee
	if err := r.db.WithContext(ctx).Where("employment_status = ? AND department IN (?)", "Active", departmentNames).Find(&employees).Error; err != nil {
		return nil, err
	}
	return employees, nil
}

// FindActiveByEmployeeIDs 查询指定员工业务工号列表中的所有在职员工
func (r *gormEmployeeRepository) FindActiveByEmployeeIDs(ctx context.Context, employeeIDs []string) ([]models.Employee, error) {
	var employees []models.Employee
	if err := r.db.WithContext(ctx).Where("employment_status = ? AND employee_id IN (?)", "Active", employeeIDs).Find(&employees).Error; err != nil {
		return nil, err
	}
	return employees, nil
}
