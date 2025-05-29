package services

import (
	"context"
	"errors"
	"time"

	// "unicode" // 移除 unicode, isNumeric 已移到 utils

	// "github.com/phone_management/internal/handlers" // 移除此导入
	"github.com/phone_management/internal/models"
	"github.com/phone_management/internal/repositories"
	"github.com/phone_management/pkg/utils" // 导入 utils 包
)

// ErrEmployeeNotFound 表示员工未找到的错误 (虽然创建时不用，但通常服务层会有)
var ErrEmployeeNotFound = errors.New("员工未找到")

// ErrPhoneNumberExists 表示手机号码已存在 (服务层错误)
var ErrPhoneNumberExists = errors.New("手机号码已存在")

// ErrEmailExists 表示邮箱已存在 (服务层错误)
var ErrEmailExists = errors.New("邮箱已存在")

var ErrEmployeeNameNotFound = errors.New("按姓名未找到员工记录")

// 员工离职相关错误
var ErrEmployeeHasActiveNumbers = errors.New("员工当前正在使用手机号码，无法办理离职")

// EmployeeService 定义了员工服务的接口
type EmployeeService interface {
	CreateEmployee(employee *models.Employee) (*models.Employee, error)
	GetEmployees(page, limit int, sortBy, sortOrder, search, employmentStatus string) ([]models.Employee, int64, error)
	GetEmployeeDetailByEmployeeID(employeeID string) (*models.EmployeeDetailResponse, error)
	GetEmployeeByEmployeeID(employeeID string) (*models.Employee, error)
	UpdateEmployee(employeeID string, payload models.UpdateEmployeePayload) (*models.Employee, error)
	GetEmployeesByFullName(fullName string) ([]*models.Employee, error)
}

// employeeService 是 EmployeeService 的实现
type employeeService struct {
	repo             repositories.EmployeeRepository
	mobileNumberRepo repositories.MobileNumberRepository
}

// NewEmployeeService 创建一个新的 employeeService 实例
func NewEmployeeService(repo repositories.EmployeeRepository, mobileNumberRepo repositories.MobileNumberRepository) EmployeeService {
	return &employeeService{
		repo:             repo,
		mobileNumberRepo: mobileNumberRepo,
	}
}

// CreateEmployee 处理创建员工的业务逻辑
func (s *employeeService) CreateEmployee(employee *models.Employee) (*models.Employee, error) {
	if employee.PhoneNumber != nil && *employee.PhoneNumber != "" {
		phone := *employee.PhoneNumber
		// 使用 utils 中的校验函数
		if err := utils.ValidatePhoneNumber(phone); err != nil {
			return nil, err // 直接返回 utils 包中定义的错误 (ErrInvalidPhoneNumberFormat 或 ErrInvalidPhoneNumberPrefix)
		}

		// 唯一性校验
		_, err := s.repo.GetEmployeeByPhoneNumber(phone)
		if err == nil {
			return nil, ErrPhoneNumberExists // 服务层特定的唯一性冲突错误
		} else if !errors.Is(err, repositories.ErrRecordNotFound) {
			return nil, err
		}
	}

	// 邮箱唯一性校验 (格式校验由 handler 层或 model binding 处理，服务层主要关注唯一性)
	if employee.Email != nil && *employee.Email != "" {
		// 格式校验如果需要在此处加强，也可以调用 utils.ValidateEmailFormat(*employee.Email)
		// 但当前批量导入已在handler层校验，单个创建依赖Gin的binding。为保持一致，此处主要负责唯一性。
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
	currentEmployee, err := s.repo.GetEmployeeByEmployeeID(employeeID)
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

	if payload.HireDate != nil {
		if *payload.HireDate == "" {
			updates["hire_date"] = nil
		} else {
			hireDate, err := time.Parse("2006-01-02", *payload.HireDate)
			if err != nil {
				return nil, errors.New("无效的入职日期格式: " + *payload.HireDate)
			}
			updates["hire_date"] = &hireDate
		}
	}

	statusUpdated := false
	if payload.EmploymentStatus != nil {
		// 如果要将员工状态更新为"Departed"，需要进行离职检查
		if *payload.EmploymentStatus == "Departed" && currentEmployee.EmploymentStatus != "Departed" {
			// 执行离职前检查和处理
			if err := s.handleEmployeeDeparture(employeeID); err != nil {
				return nil, err
			}
		}

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

// handleEmployeeDeparture 处理员工离职时的业务逻辑
func (s *employeeService) handleEmployeeDeparture(employeeID string) error {
	ctx := context.Background()

	// 1. 检查员工是否有使用中的手机号码（状态为 in_use）
	assignedNumbers, err := s.mobileNumberRepo.FindAssignedToEmployee(ctx, employeeID)
	if err != nil {
		return err
	}

	// 过滤出状态为"使用中"的号码
	var activeNumbers []models.MobileNumber
	for _, number := range assignedNumbers {
		if number.Status == string(models.StatusInUse) {
			activeNumbers = append(activeNumbers, number)
		}
	}

	// 如果有使用中的号码，拒绝离职
	if len(activeNumbers) > 0 {
		return ErrEmployeeHasActiveNumbers
	}

	// 2. 查找该员工作为办卡人的所有号码
	applicantNumbers, err := s.mobileNumberRepo.FindByApplicantEmployeeID(ctx, employeeID)
	if err != nil {
		return err
	}

	// 3. 将办卡人的号码状态更新为 risk_pending（除了已经是终止状态的）
	var numberIDsToUpdate []uint
	for _, number := range applicantNumbers {
		// 只更新非终止状态的号码
		if number.Status != string(models.StatusDeactivated) &&
			number.Status != string(models.StatusRiskPending) &&
			number.Status != string(models.StatusUserReport) {
			numberIDsToUpdate = append(numberIDsToUpdate, number.ID)
		}
	}

	// 批量更新号码状态
	if len(numberIDsToUpdate) > 0 {
		if err := s.mobileNumberRepo.BatchUpdateStatus(ctx, numberIDsToUpdate, string(models.StatusRiskPending)); err != nil {
			return err
		}
	}

	return nil
}
