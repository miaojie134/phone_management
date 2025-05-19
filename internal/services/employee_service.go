package services

import (
	"errors"
	"strings"
	"time"
	"unicode"

	// "github.com/phone_management/internal/handlers" // 移除此导入
	"github.com/phone_management/internal/models"
	"github.com/phone_management/internal/repositories"
)

// ErrEmployeeNotFound 表示员工未找到的错误 (虽然创建时不用，但通常服务层会有)
var ErrEmployeeNotFound = errors.New("员工未找到")

// ErrPhoneNumberExists 表示手机号码已存在 (服务层错误)
var ErrPhoneNumberExists = errors.New("手机号码已存在")

// ErrEmailExists 表示邮箱已存在 (服务层错误)
var ErrEmailExists = errors.New("邮箱已存在")

// ErrInvalidPhoneNumberFormat 表示手机号码格式不正确 (例如非纯数字、长度不对等)
var ErrInvalidPhoneNumberFormat = errors.New("无效的手机号码格式")

// ErrInvalidPhoneNumberPrefix 表示手机号码前缀不正确 (例如不是以'1'开头)
var ErrInvalidPhoneNumberPrefix = errors.New("无效的手机号码前缀，必须以1开头")

var ErrEmployeeNameNotFound = errors.New("按姓名未找到员工记录") // 新增错误

// isNumeric 辅助函数，检查字符串是否只包含数字
func isNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// EmployeeService 定义了员工服务的接口
type EmployeeService interface {
	CreateEmployee(employee *models.Employee) (*models.Employee, error)
	GetEmployees(page, limit int, sortBy, sortOrder, search, employmentStatus string) ([]models.Employee, int64, error)
	GetEmployeeDetailByEmployeeID(employeeID string) (*models.EmployeeDetailResponse, error)
	GetEmployeeByEmployeeID(employeeID string) (*models.Employee, error)
	UpdateEmployee(employeeID string, payload models.UpdateEmployeePayload) (*models.Employee, error)
	GetEmployeesByFullName(fullName string) ([]*models.Employee, error) // 新增方法
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
	// 手机号码校验 (如果提供了手机号)
	if employee.PhoneNumber != nil && *employee.PhoneNumber != "" {
		phone := *employee.PhoneNumber

		// 1. 长度必须为11位
		if len(phone) != 11 {
			return nil, ErrInvalidPhoneNumberFormat // 或者更具体的长度错误，但格式错误已包含此意
		}
		// 2. 必须全部是数字
		if !isNumeric(phone) { // 使用辅助函数
			return nil, ErrInvalidPhoneNumberFormat
		}
		// 3. 必须以数字 '1' 开头
		if !strings.HasPrefix(phone, "1") {
			return nil, ErrInvalidPhoneNumberPrefix
		}

		// 唯一性校验 (已存在逻辑)
		_, err := s.repo.GetEmployeeByPhoneNumber(phone)
		if err == nil {
			return nil, ErrPhoneNumberExists
		} else if !errors.Is(err, repositories.ErrRecordNotFound) {
			return nil, err
		}
	}

	// 邮箱唯一性校验 (已存在逻辑)
	if employee.Email != nil && *employee.Email != "" {
		_, err := s.repo.GetEmployeeByEmail(*employee.Email)
		if err == nil {
			return nil, ErrEmailExists
		} else if !errors.Is(err, repositories.ErrRecordNotFound) {
			return nil, err
		}
	}

	if employee.EmploymentStatus == "" {
		employee.EmploymentStatus = "Active"
	}

	createdEmployee, err := s.repo.CreateEmployee(employee)
	if err != nil {
		if errors.Is(err, repositories.ErrEmployeePhoneNumberConflict) {
			return nil, ErrPhoneNumberExists
		}
		if errors.Is(err, repositories.ErrEmployeeEmailConflict) {
			return nil, ErrEmailExists
		}
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

// GetEmployeeDetailByEmployeeID 处理根据业务工号获取员工详情的业务逻辑
func (s *employeeService) GetEmployeeDetailByEmployeeID(employeeID string) (*models.EmployeeDetailResponse, error) {
	employeeDetail, err := s.repo.GetEmployeeDetailByEmployeeID(employeeID)
	if err != nil {
		if errors.Is(err, repositories.ErrRecordNotFound) {
			return nil, ErrEmployeeNotFound // 转为服务层定义的错误
		}
		return nil, err
	}
	return employeeDetail, nil
}

// GetEmployeeByEmployeeID 处理根据业务工号获取员工的业务逻辑
func (s *employeeService) GetEmployeeByEmployeeID(employeeID string) (*models.Employee, error) {
	employee, err := s.repo.GetEmployeeByEmployeeID(employeeID)
	if err != nil {
		if errors.Is(err, repositories.ErrRecordNotFound) {
			return nil, ErrEmployeeNotFound // 转为服务层定义的错误
		}
		return nil, err
	}
	return employee, nil
}

// GetEmployeesByFullName 根据员工全名查找员工
func (s *employeeService) GetEmployeesByFullName(fullName string) ([]*models.Employee, error) {
	employees, err := s.repo.GetEmployeesByFullName(fullName)
	if err != nil {
		return nil, err // 直接返回仓库层错误
	}
	if len(employees) == 0 {
		return nil, ErrEmployeeNameNotFound // 如果列表为空，返回特定错误
	}
	return employees, nil
}

// UpdateEmployee 处理更新员工信息的业务逻辑
func (s *employeeService) UpdateEmployee(employeeID string, payload models.UpdateEmployeePayload) (*models.Employee, error) {
	// 首先，确保员工存在
	_, err := s.repo.GetEmployeeByEmployeeID(employeeID)
	if err != nil {
		if errors.Is(err, repositories.ErrRecordNotFound) {
			return nil, ErrEmployeeNotFound
		}
		return nil, err
	}

	updates := make(map[string]interface{})

	if payload.Department != nil {
		updates["department"] = *payload.Department
	}

	statusUpdated := false
	if payload.EmploymentStatus != nil {
		updates["employment_status"] = *payload.EmploymentStatus
		statusUpdated = true
		// 如果状态更新为 "Departed"
		if *payload.EmploymentStatus == "Departed" {
			if payload.TerminationDate != nil && *payload.TerminationDate != "" {
				termDate, err := time.Parse("2006-01-02", *payload.TerminationDate)
				if err != nil {
					return nil, errors.New("无效的离职日期格式: " + *payload.TerminationDate)
				}
				updates["termination_date"] = &termDate
			} else {
				// 如果状态是 Departed 但 payload 中没有提供离职日期，则自动设置为当前日期
				now := time.Now()
				updates["termination_date"] = &now
			}
		} else {
			// 如果状态更新为非 "Departed" (e.g., "Active", "Inactive"), 则应清除离职日期
			updates["termination_date"] = nil
		}
	}

	if !statusUpdated && payload.TerminationDate != nil {
		if *payload.TerminationDate == "" {
			updates["termination_date"] = nil
		} else {
			termDate, err := time.Parse("2006-01-02", *payload.TerminationDate)
			if err != nil {
				return nil, errors.New("无效的离职日期格式: " + *payload.TerminationDate)
			}
			updates["termination_date"] = &termDate
		}
	}

	if len(updates) == 0 {
		return nil, errors.New("没有提供任何有效的更新字段")
	}

	return s.repo.UpdateEmployee(employeeID, updates)
}
