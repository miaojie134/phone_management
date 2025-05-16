package services

import (
	"errors"
	"time"

	"github.com/phone_management/internal/models"
	"github.com/phone_management/internal/repositories"
)

// ErrMobileNumberNotFound 表示手机号码未找到的错误
var ErrMobileNumberNotFound = errors.New("手机号码未找到")

// MobileNumberService 定义了手机号码服务的接口
type MobileNumberService interface {
	// CreateMobileNumber 的 mobileNumber 参数中已包含 ApplicantEmployeeID (string)
	CreateMobileNumber(mobileNumber *models.MobileNumber) (*models.MobileNumber, error)
	GetMobileNumbers(page, limit int, sortBy, sortOrder, search, status, applicantStatus string) ([]models.MobileNumberResponse, int64, error)
	GetMobileNumberByID(id uint) (*models.MobileNumberResponse, error)
	UpdateMobileNumber(id uint, payload models.MobileNumberUpdatePayload) (*models.MobileNumber, error)
	// AssignMobileNumber 的 employeeBusinessID 参数是 string (业务工号)
	AssignMobileNumber(numberID uint, employeeBusinessID string, assignmentDate time.Time) (*models.MobileNumber, error)
	UnassignMobileNumber(numberID uint, reclaimDate time.Time) (*models.MobileNumber, error)
}

// mobileNumberService 是 MobileNumberService 的实现
type mobileNumberService struct {
	repo            repositories.MobileNumberRepository
	employeeService EmployeeService // 使用接口类型 EmployeeService，而不是 services.EmployeeService
}

// NewMobileNumberService 创建一个新的 mobileNumberService 实例
func NewMobileNumberService(repo repositories.MobileNumberRepository, empService EmployeeService) MobileNumberService { // 参数类型也改为接口类型
	return &mobileNumberService{repo: repo, employeeService: empService}
}

// CreateMobileNumber 处理创建手机号码的业务逻辑
// mobileNumber.ApplicantEmployeeID (string) 已经由 handler 层从 payload 设置
func (s *mobileNumberService) CreateMobileNumber(mobileNumber *models.MobileNumber) (*models.MobileNumber, error) {
	// 1. 验证 ApplicantEmployeeID (员工业务工号) 是否有效 (即员工是否存在)
	_, err := s.employeeService.GetEmployeeByBusinessID(mobileNumber.ApplicantEmployeeID)
	if err != nil {
		// err 可能是 ErrEmployeeNotFound 或其他DB错误
		// 如果是 ErrEmployeeNotFound，handler 层会捕获并返回 404
		return nil, err
	}
	// (可选) 若将来需要，可在此处用上面查询到的 employee 对象做进一步校验，例如：
	// if queriedApplicant.EmploymentStatus != "Active" { ... }

	// ApplicantEmployeeID (string) 已在 mobileNumber 对象中，直接传递给仓库层创建
	createdMobileNumber, err := s.repo.CreateMobileNumber(mobileNumber)
	if err != nil {
		return nil, err
	}
	return createdMobileNumber, nil
}

// GetMobileNumbers 处理获取手机号码列表的业务逻辑
func (s *mobileNumberService) GetMobileNumbers(page, limit int, sortBy, sortOrder, search, status, applicantStatus string) ([]models.MobileNumberResponse, int64, error) {
	// 当前业务逻辑主要是参数传递和调用仓库层
	// 未来可在这里添加更复杂的业务规则
	return s.repo.GetMobileNumbers(page, limit, sortBy, sortOrder, search, status, applicantStatus)
}

// GetMobileNumberByID 处理根据ID获取手机号码详情的业务逻辑
func (s *mobileNumberService) GetMobileNumberByID(id uint) (*models.MobileNumberResponse, error) {
	mobileNumber, err := s.repo.GetMobileNumberByID(id)
	if err != nil {
		// 如果仓库层返回 gorm.ErrRecordNotFound，则转换为业务层定义的 ErrMobileNumberNotFound
		if errors.Is(err, repositories.ErrRecordNotFound) { // 假设 repo 层会返回这个 gorm 标准错误或自定义错误
			return nil, ErrMobileNumberNotFound
		}
		return nil, err
	}
	return mobileNumber, nil
}

// UpdateMobileNumber 处理更新手机号码的业务逻辑
func (s *mobileNumberService) UpdateMobileNumber(id uint, payload models.MobileNumberUpdatePayload) (*models.MobileNumber, error) {
	updates := make(map[string]interface{})

	if payload.Status != nil {
		updates["status"] = *payload.Status
		// 当号码状态变更为"已注销"时，自动记录注销时间。
		if *payload.Status == string(models.StatusDeactivated) {
			now := time.Now()
			updates["cancellation_date"] = &now
		}
	}
	if payload.Vendor != nil {
		updates["vendor"] = *payload.Vendor
	}
	if payload.Remarks != nil {
		updates["remarks"] = *payload.Remarks
	}

	if len(updates) == 0 {
		// 如果没有提供任何要更新的字段，可以返回一个错误或直接返回未修改的记录
		// 这里选择返回错误，因为API期望至少更新一个字段
		return nil, errors.New("没有提供任何更新字段")
	}

	updatedMobileNumber, err := s.repo.UpdateMobileNumber(id, updates)
	if err != nil {
		if errors.Is(err, repositories.ErrRecordNotFound) {
			return nil, ErrMobileNumberNotFound
		}
		return nil, err
	}

	return updatedMobileNumber, nil
}

// AssignMobileNumber 处理将手机号码分配给员工的业务逻辑
// employeeBusinessID 是员工的业务工号 (string)
func (s *mobileNumberService) AssignMobileNumber(numberID uint, employeeBusinessID string, assignmentDate time.Time) (*models.MobileNumber, error) {
	// 1. 验证 employeeBusinessID (员工业务工号) 是否有效且在职
	assignee, err := s.employeeService.GetEmployeeByBusinessID(employeeBusinessID)
	if err != nil {
		return nil, err // err 可能是 ErrEmployeeNotFound 或其他DB错误
	}
	if assignee.EmploymentStatus != "Active" { // 确保员工在职才能分配号码
		return nil, repositories.ErrEmployeeNotActive // 复用仓库层的错误，表示员工非在职
	}

	// employeeBusinessID (string) 直接传递给仓库层
	// 仓库层 AssignMobileNumber 内部还会再次查询员工以确认状态，这可以视为一种双重保障或允许仓库层独立校验。
	// 如果希望避免仓库层重复查询员工（因为这里已经查过），可以调整仓库层 AssignMobileNumber 的逻辑。
	// 但当前保持不变，让仓库层也进行校验。
	assignedMobileNumber, err := s.repo.AssignMobileNumber(numberID, employeeBusinessID, assignmentDate)
	if err != nil {
		if errors.Is(err, repositories.ErrRecordNotFound) {
			// 此处的 ErrRecordNotFound 是针对 MobileNumber 的，由 repo.AssignMobileNumber 返回
			return nil, ErrMobileNumberNotFound
		}
		// 其他特定错误如 ErrMobileNumberNotInIdleStatus, ErrEmployeeNotFound (如果仓库层校验员工失败)
		// ErrEmployeeNotActive (如果仓库层校验员工状态失败) 会直接从 repo 传递上来。
		return nil, err
	}
	return assignedMobileNumber, nil
}

// UnassignMobileNumber 处理从当前用户回收手机号码的业务逻辑
func (s *mobileNumberService) UnassignMobileNumber(numberID uint, reclaimDate time.Time) (*models.MobileNumber, error) {
	unassignedMobileNumber, err := s.repo.UnassignMobileNumber(numberID, reclaimDate)
	if err != nil {
		// 错误转换：将仓库层特定的错误转换为服务层或通用的错误
		if errors.Is(err, repositories.ErrRecordNotFound) { // 号码未找到
			return nil, ErrMobileNumberNotFound
		}
		// 其他特定错误如 ErrMobileNumberNotInUseStatus, ErrNoActiveUsageHistoryFound 会直接从 repo 传递上来
		return nil, err
	}
	return unassignedMobileNumber, nil
}
