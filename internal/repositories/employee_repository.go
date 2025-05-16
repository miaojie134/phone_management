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
	// 预先检查 employeeId 是否已存在
	var existing models.Employee
	if err := r.db.Unscoped().Where("employee_id = ?", employee.EmployeeID).First(&existing).Error; err == nil {
		// 如果找到了记录（即使是软删除的），也认为工号已存在，以防止恢复时冲突或业务逻辑混乱
		return nil, ErrEmployeeIDExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 如果是其他查询错误
		return nil, err
	}

	// 如果记录未找到，则创建新记录
	if err := r.db.Create(employee).Error; err != nil {
		// GORM 通常会将数据库的唯一约束违例错误包装起来
		// 对于 SQLite，错误信息可能包含 "UNIQUE constraint failed"
		if strings.Contains(strings.ToLower(err.Error()), "unique constraint") || strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			if strings.Contains(err.Error(), "employees.employee_id") { // 更精确地判断是 employee_id 的唯一约束
				return nil, ErrEmployeeIDExists
			}
		}
		return nil, err
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
