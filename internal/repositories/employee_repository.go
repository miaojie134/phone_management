package repositories

import (
	"errors"
	"strings"

	"github.com/phone_management/internal/models"
	"gorm.io/gorm"
)

// ErrEmployeeIDExists 表示员工工号已存在
var ErrEmployeeIDExists = errors.New("员工工号已存在")

// EmployeeRepository 定义了员工数据仓库的接口
type EmployeeRepository interface {
	CreateEmployee(employee *models.Employee) (*models.Employee, error)
	GetEmployees(page, limit int, sortBy, sortOrder, search, employmentStatus string) ([]models.Employee, int64, error)
	GetEmployeeDetailByEmployeeID(employeeID string) (*models.EmployeeDetailResponse, error)
	GetEmployeeByEmployeeID(employeeID string) (*models.Employee, error)
	UpdateEmployee(employeeID string, updates map[string]interface{}) (*models.Employee, error)
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
	// 由于 EmployeeID 现在由系统生成（在服务层完成赋值），
	// 此处不再需要预先检查 EmployeeID 是否已存在。
	// 唯一性将由数据库的 UNIQUE 约束来保证。
	// 如果生成算法出现碰撞（极小概率），数据库的 Create 操作会失败并返回错误。

	// var existing models.Employee
	// if err := r.db.Unscoped().Where("employee_id = ?", employee.EmployeeID).First(&existing).Error; err == nil {
	// 	return nil, ErrEmployeeIDExists
	// } else if !errors.Is(err, gorm.ErrRecordNotFound) {
	// 	return nil, err
	// }

	// 直接创建新记录
	if err := r.db.Create(employee).Error; err != nil {
		// GORM 通常会将数据库的唯一约束违例错误包装起来
		// 例如，对于 SQLite，错误信息可能包含 "UNIQUE constraint failed: employees.employee_id"
		// 对于 MySQL，可能是 "Error 1062: Duplicate entry 'some_employee_id' for key 'employee_id_unique_constraint_name'"
		// 我们可以检查这个错误是否与 employee_id 的唯一性有关，并返回一个更具体的错误，
		// 但为了保持仓库层的通用性，通常直接返回数据库错误，由服务层或 handler 层决定如何解释和响应。
		// 如果需要更精细的控制，可以解析 err.Error() 字符串，但这可能因数据库类型而异。
		// 或者，GORM 可能提供更结构化的方式来识别特定类型的约束违例，但这需要查阅GORM文档。
		// 暂时，我们依赖 ErrEmployeeIDExists（如果它仍然被服务层或handler层使用并期望）。
		// 但由于我们这里不再主动检查并返回 ErrEmployeeIDExists，如果上层还依赖它，需要调整。
		// 当前服务层 CreateEmployee 的错误处理是直接向上传递 err。
		// handler 层会捕获 repositories.ErrEmployeeIDExists，但这个错误现在不会从这里发出。
		// handler 层需要调整为捕获更通用的数据库错误，或者服务层需要转换这个错误。

		// 为了保持与之前 handler 行为的一致性（当工号冲突时返回 409 Conflict），
		// 我们在这里可以尝试判断是否是 employee_id 的唯一约束错误。
		if strings.Contains(strings.ToLower(err.Error()), "unique constraint") && strings.Contains(strings.ToLower(err.Error()), "employee_id") {
			return nil, ErrEmployeeIDExists // 复用之前的错误类型，以便handler可以正确处理
		}
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") && strings.Contains(strings.ToLower(err.Error()), "employee_id") { // PostgreSQL, SQL Server
			return nil, ErrEmployeeIDExists
		}
		if strings.Contains(strings.ToLower(err.Error()), "duplicate entry") && strings.Contains(strings.ToLower(err.Error()), "employees.employee_id") { // MySQL like
			return nil, ErrEmployeeIDExists
		}
		// ... 其他数据库的唯一约束错误检查

		return nil, err // 返回原始错误，如果不是已知的 employee_id 唯一约束冲突
	}
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
