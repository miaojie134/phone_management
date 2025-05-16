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
	CreateMobileNumber(mobileNumber *models.MobileNumber) (*models.MobileNumber, error)
	GetMobileNumbers(page, limit int, sortBy, sortOrder, search, status, applicantStatus string) ([]models.MobileNumberResponse, int64, error)
	GetMobileNumberByID(id uint) (*models.MobileNumberResponse, error)
	UpdateMobileNumber(id uint, payload models.MobileNumberUpdatePayload) (*models.MobileNumber, error)
	AssignMobileNumber(numberID uint, employeeID uint, assignmentDate time.Time) (*models.MobileNumber, error)
	UnassignMobileNumber(numberID uint, reclaimDate time.Time) (*models.MobileNumber, error)
}

// mobileNumberService 是 MobileNumberService 的实现
type mobileNumberService struct {
	repo repositories.MobileNumberRepository
	// employeeRepo repositories.EmployeeRepository // 未来可注入员工仓库用于校验 ApplicantEmployeeDbID
}

// NewMobileNumberService 创建一个新的 mobileNumberService 实例
func NewMobileNumberService(repo repositories.MobileNumberRepository) MobileNumberService {
	return &mobileNumberService{repo: repo}
}

// CreateMobileNumber 处理创建手机号码的业务逻辑
func (s *mobileNumberService) CreateMobileNumber(mobileNumber *models.MobileNumber) (*models.MobileNumber, error) {
	// 当前业务逻辑比较简单，直接调用仓库层
	// 未来可在这里添加更复杂的业务规则，例如：
	// 1. 检查办卡人 ID (ApplicantEmployeeDbID) 是否有效 (需要 EmployeeRepository)
	// 2. 根据特定规则自动生成某些字段的值等

	createdMobileNumber, err := s.repo.CreateMobileNumber(mobileNumber)
	if err != nil {
		return nil, err // 将仓库层错误（包括 ErrPhoneNumberExists）直接向上传递
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
func (s *mobileNumberService) AssignMobileNumber(numberID uint, employeeID uint, assignmentDate time.Time) (*models.MobileNumber, error) {
	assignedMobileNumber, err := s.repo.AssignMobileNumber(numberID, employeeID, assignmentDate)
	if err != nil {
		// 错误转换：将仓库层特定的错误转换为服务层或通用的错误
		if errors.Is(err, repositories.ErrRecordNotFound) { // 号码或员工未找到（在仓库层AssignMobileNumber中会区分返回ErrEmployeeNotFound或ErrRecordNotFound for number）
			// 决定是返回更具体的错误还是统一的"未找到"
			// 为了清晰，这里可以考虑不在service层再转换 ErrEmployeeNotFound，直接透传
			// 但如果 ErrRecordNotFound 是针对 mobile number 的，则转换为 ErrMobileNumberNotFound
			// 鉴于 repo.AssignMobileNumber 返回的 ErrRecordNotFound 是针对 MobileNumber 的，所以转换为 ErrMobileNumberNotFound
			// 而 ErrEmployeeNotFound, ErrMobileNumberNotInIdleStatus, ErrEmployeeNotActive 会直接从repo透传过来
			return nil, ErrMobileNumberNotFound
		}
		// 其他特定错误如 ErrMobileNumberNotInIdleStatus, ErrEmployeeNotFound, ErrEmployeeNotActive 会直接从 repo 传递上来
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
