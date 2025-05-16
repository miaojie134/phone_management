package services

import (
	"errors"

	"github.com/phone_management/internal/models"
	"github.com/phone_management/internal/repositories"
)

// ErrEmployeeNotFound 表示员工未找到的错误 (虽然创建时不用，但通常服务层会有)
var ErrEmployeeNotFound = errors.New("员工未找到")

// EmployeeService 定义了员工服务的接口
type EmployeeService interface {
	CreateEmployee(employee *models.Employee) (*models.Employee, error)
	GetEmployees(page, limit int, sortBy, sortOrder, search, employmentStatus string) ([]models.Employee, int64, error)
}

// employeeService 是 EmployeeService 的实现
type employeeService struct {
	repo repositories.EmployeeRepository
}

// NewEmployeeService 创建一个新的 employeeService 实例
func NewEmployeeService(repo repositories.EmployeeRepository) EmployeeService {
	return &employeeService{repo: repo}
}

// CreateEmployee 处理创建员工的业务逻辑
func (s *employeeService) CreateEmployee(employee *models.Employee) (*models.Employee, error) {
	// 可以在这里添加更复杂的业务规则，例如默认值设置等
	// 根据文档，employmentStatus 默认为 "Active"，这应该由模型或数据库层面处理，但也可在此校验或强制设置
	if employee.EmploymentStatus == "" {
		employee.EmploymentStatus = "Active" // 如果模型中没有默认值，这里可以补上
	}

	createdEmployee, err := s.repo.CreateEmployee(employee)
	if err != nil {
		// 错误可以直接向上传递，handler 层会根据错误类型进行不同响应
		return nil, err
	}
	return createdEmployee, nil
}

// GetEmployees 处理获取员工列表的业务逻辑
func (s *employeeService) GetEmployees(page, limit int, sortBy, sortOrder, search, employmentStatus string) ([]models.Employee, int64, error) {
	// 当前业务逻辑主要是参数传递和调用仓库层
	// 未来可在这里添加更复杂的业务规则，如数据转换或权限校验等
	return s.repo.GetEmployees(page, limit, sortBy, sortOrder, search, employmentStatus)
}
