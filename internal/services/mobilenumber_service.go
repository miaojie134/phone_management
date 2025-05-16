package services

import (
	"github.com/phone_management/internal/models"
	"github.com/phone_management/internal/repositories"
)

// MobileNumberService 定义了手机号码服务的接口
type MobileNumberService interface {
	CreateMobileNumber(mobileNumber *models.MobileNumber) (*models.MobileNumber, error)
	GetMobileNumbers(page, limit int, sortBy, sortOrder, search, status, applicantStatus string) ([]models.MobileNumberResponse, int64, error)
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
