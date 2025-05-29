package services

import (
	"context"
	"errors"
	"time"

	"github.com/phone_management/internal/models"
	"github.com/phone_management/internal/repositories"
	"github.com/phone_management/pkg/utils"
)

// ErrMobileNumberNotFound 表示手机号码未找到的错误
var ErrMobileNumberNotFound = errors.New("手机号码未找到")

// 错误定义
var ErrApplicantNameNotFound = errors.New("办卡人姓名未找到")
var ErrApplicantNameNotUnique = errors.New("办卡人姓名存在重名，无法唯一确定员工，请在系统中确保该姓名唯一或联系管理员处理")

// MobileNumberService 定义了手机号码服务的接口
type MobileNumberService interface {
	// CreateMobileNumber 的 mobileNumber 参数中已包含 ApplicantEmployeeID (string)
	CreateMobileNumber(mobileNumber *models.MobileNumber) (*models.MobileNumber, error)
	GetMobileNumbers(page, limit int, sortBy, sortOrder, search, status, applicantStatus string) ([]models.MobileNumberResponse, int64, error)
	GetMobileNumberByPhoneNumberDetail(phoneNumber string) (*models.MobileNumberResponse, error)
	UpdateMobileNumberByPhoneNumber(phoneNumber string, payload models.MobileNumberUpdatePayload) (*models.MobileNumber, error)
	// AssignMobileNumber 的 employeeBusinessID 参数是 string (业务工号)
	// 第一个参数从 numberID uint 修改为 phoneNumber string
	AssignMobileNumber(phoneNumber string, employeeBusinessID string, assignmentDate time.Time, purpose string) (*models.MobileNumber, error)
	// UnassignMobileNumber(numberID uint, reclaimDate time.Time) (*models.MobileNumber, error) // 旧方法
	UnassignMobileNumberByPhoneNumber(phoneNumber string, reclaimDate time.Time) (*models.MobileNumber, error) //
	ResolveApplicantNameToID(applicantName string) (string, error)                                             //
	// 风险号码处理相关方法
	GetRiskPendingNumbers(page, limit int, sortBy, sortOrder, search, applicantStatus string) ([]models.RiskNumberResponse, int64, error)
	HandleRiskNumber(phoneNumber string, payload models.HandleRiskNumberPayload, operatorEmployeeID string) (*models.MobileNumber, error)
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
	// 0. 校验手机号码格式 (使用 utils 中的校验函数)
	if err := utils.ValidatePhoneNumber(mobileNumber.PhoneNumber); err != nil {
		return nil, err // 直接返回 utils 包中定义的错误
	}

	// 1. 验证 ApplicantEmployeeID (员工业务工号) 是否有效 (即员工是否存在)
	_, err := s.employeeService.GetEmployeeByEmployeeID(mobileNumber.ApplicantEmployeeID)
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

// GetMobileNumberByPhoneNumberDetail 处理根据手机号码字符串获取手机号码详情的业务逻辑
func (s *mobileNumberService) GetMobileNumberByPhoneNumberDetail(phoneNumber string) (*models.MobileNumberResponse, error) {
	mobileNumberDetail, err := s.repo.GetMobileNumberResponseByPhoneNumber(phoneNumber) // 假设 repo 有此方法
	if err != nil {
		if errors.Is(err, repositories.ErrRecordNotFound) {
			return nil, ErrMobileNumberNotFound
		}
		return nil, err
	}
	return mobileNumberDetail, nil
}

// UpdateMobileNumberByPhoneNumber 处理根据手机号码字符串更新手机号码的业务逻辑
func (s *mobileNumberService) UpdateMobileNumberByPhoneNumber(phoneNumber string, payload models.MobileNumberUpdatePayload) (*models.MobileNumber, error) {
	// 0. 通过 phoneNumber 获取 MobileNumber 实体及其 ID
	mobileNumber, err := s.repo.GetMobileNumberByPhoneNumber(phoneNumber)
	if err != nil {
		if errors.Is(err, repositories.ErrRecordNotFound) {
			return nil, ErrMobileNumberNotFound
		}
		return nil, err // 其他数据库错误
	}

	updates := make(map[string]interface{})
	if payload.Status != nil {
		// 验证状态是否有效
		if !models.IsValidStatus(*payload.Status) {
			return nil, errors.New("无效的状态值")
		}

		// 业务规则：不能直接更新为"使用中"状态
		if *payload.Status == string(models.StatusInUse) {
			return nil, errors.New("不能直接将状态更新为'使用中'，请使用分配操作")
		}

		// 业务规则：如果当前状态是"使用中"，不能直接更新状态
		if mobileNumber.Status == string(models.StatusInUse) {
			return nil, errors.New("号码当前为'使用中'状态，不能直接更新状态，请先进行回收操作")
		}

		updates["status"] = *payload.Status
		if *payload.Status == string(models.StatusDeactivated) {
			now := time.Now()
			updates["cancellation_date"] = &now
		}
	}
	if payload.Purpose != nil {
		updates["purpose"] = *payload.Purpose
	}
	if payload.Vendor != nil {
		updates["vendor"] = *payload.Vendor
	}
	if payload.Remarks != nil {
		updates["remarks"] = *payload.Remarks
	}

	if len(updates) == 0 {
		return nil, errors.New("没有提供任何更新字段")
	}

	// 使用获取到的 mobileNumber.ID 进行更新
	updatedMobileNumber, err := s.repo.UpdateMobileNumber(mobileNumber.ID, updates)
	if err != nil {
		// repo.UpdateMobileNumber 内部会处理 ErrRecordNotFound，这里不需要再次转换
		// 如果发生，通常意味着在 GetMobileNumberByPhoneNumber 和 UpdateMobileNumber 之间记录被删除，是一个竞争条件
		return nil, err
	}
	return updatedMobileNumber, nil
}

// AssignMobileNumber 处理将手机号码分配给员工的业务逻辑
// employeeBusinessID 是员工的业务工号 (string)
// 第一个参数从 numberID uint 修改为 phoneNumber string
func (s *mobileNumberService) AssignMobileNumber(phoneNumber string, employeeBusinessID string, assignmentDate time.Time, purpose string) (*models.MobileNumber, error) {
	// 0. 通过 phoneNumber 获取 MobileNumber 实体及其 ID
	mobileNumber, err := s.repo.GetMobileNumberByPhoneNumber(phoneNumber)
	if err != nil {
		if errors.Is(err, repositories.ErrRecordNotFound) {
			return nil, ErrMobileNumberNotFound
		}
		return nil, err // 其他数据库错误
	}

	// 1. 验证 employeeBusinessID (员工业务工号) 是否有效且在职
	assignee, err := s.employeeService.GetEmployeeByEmployeeID(employeeBusinessID)
	if err != nil {
		return nil, err // err 可能是 ErrEmployeeNotFound 或其他DB错误
	}
	if assignee.EmploymentStatus != "Active" { // 确保员工在职才能分配号码, 直接与字符串 "Active" 比较
		return nil, repositories.ErrEmployeeNotActive // 复用仓库层的错误，表示员工非在职
	}

	// 使用从 phoneNumber 查询到的 mobileNumber.ID 进行分配，并传递用途
	assignedMobileNumber, err := s.repo.AssignMobileNumber(mobileNumber.ID, employeeBusinessID, assignmentDate, purpose)
	if err != nil {
		if errors.Is(err, repositories.ErrRecordNotFound) {
			// 此处的 ErrRecordNotFound 是针对 MobileNumber 的，由 repo.AssignMobileNumber 返回
			// 理论上，如果 GetMobileNumberByPhoneNumber 成功了，这里不应该发生 ErrRecordNotFound
			// 但为保险起见，仍做转换
			return nil, ErrMobileNumberNotFound
		}
		// 其他特定错误如 ErrMobileNumberNotInIdleStatus, ErrEmployeeNotFound (如果仓库层校验员工失败)
		// ErrEmployeeNotActive (如果仓库层校验员工状态失败) 会直接从 repo 传递上来。
		return nil, err
	}
	return assignedMobileNumber, nil
}

// UnassignMobileNumberByPhoneNumber 处理根据手机号码字符串从当前用户回收手机号码的业务逻辑
func (s *mobileNumberService) UnassignMobileNumberByPhoneNumber(phoneNumber string, reclaimDate time.Time) (*models.MobileNumber, error) {
	// 0. 通过 phoneNumber 获取 MobileNumber 实体及其 ID
	mobileNumber, err := s.repo.GetMobileNumberByPhoneNumber(phoneNumber)
	if err != nil {
		if errors.Is(err, repositories.ErrRecordNotFound) {
			return nil, ErrMobileNumberNotFound
		}
		return nil, err // 其他数据库错误
	}

	// 使用获取到的 mobileNumber.ID 进行回收
	unassignedMobileNumber, err := s.repo.UnassignMobileNumber(mobileNumber.ID, reclaimDate)
	if err != nil {
		// repo.UnassignMobileNumber 内部会处理 ErrRecordNotFound，这里不需要再次转换
		// 其他特定错误如 ErrMobileNumberNotInUseStatus, ErrNoActiveUsageHistoryFound 会直接从 repo 传递上来
		return nil, err
	}
	return unassignedMobileNumber, nil
}

// ResolveApplicantNameToID 根据办卡人姓名解析为唯一的员工业务工号
func (s *mobileNumberService) ResolveApplicantNameToID(applicantName string) (string, error) {
	employees, err := s.employeeService.GetEmployeesByFullName(applicantName)
	if err != nil {
		if errors.Is(err, ErrEmployeeNameNotFound) { // employee_service 返回的特定错误
			return "", ErrApplicantNameNotFound // 转换为 MobileNumberService 的特定错误
		}
		return "", err // 其他来自 employeeService 的错误
	}

	// 虽然 employeeService.GetEmployeesByFullName 在空列表时返回 ErrEmployeeNameNotFound，
	// 但为保险起见，这里也检查一下长度（例如，如果将来 GetEmployeesByFullName 改变了行为）
	if len(employees) == 0 {
		return "", ErrApplicantNameNotFound
	}

	if len(employees) > 1 {
		return "", ErrApplicantNameNotUnique
	}

	return employees[0].EmployeeID, nil
}

// GetRiskPendingNumbers 处理获取风险号码列表的业务逻辑
func (s *mobileNumberService) GetRiskPendingNumbers(page, limit int, sortBy, sortOrder, search, applicantStatus string) ([]models.RiskNumberResponse, int64, error) {
	// 当前业务逻辑主要是参数传递和调用仓库层
	// 未来可在这里添加更复杂的业务规则
	return s.repo.GetRiskPendingNumbers(page, limit, sortBy, sortOrder, search, applicantStatus)
}

// HandleRiskNumber 处理处理风险号码的业务逻辑
func (s *mobileNumberService) HandleRiskNumber(phoneNumber string, payload models.HandleRiskNumberPayload, operatorEmployeeID string) (*models.MobileNumber, error) {
	// 1. 验证 operatorEmployeeID (操作员业务工号) 是否有效且在职
	operator, err := s.employeeService.GetEmployeeByEmployeeID(operatorEmployeeID)
	if err != nil {
		return nil, err // err 可能是 ErrEmployeeNotFound 或其他DB错误
	}
	if operator.EmploymentStatus != "Active" { // 确保操作员在职才能处理号码, 直接与字符串 "Active" 比较
		return nil, repositories.ErrEmployeeNotActive // 复用仓库层的错误，表示操作员非在职
	}

	// 2. 调用仓库层处理风险号码
	ctx := context.Background()
	handledMobileNumber, err := s.repo.HandleRiskNumber(ctx, phoneNumber, payload, operatorEmployeeID)
	if err != nil {
		if errors.Is(err, repositories.ErrRecordNotFound) {
			return nil, ErrMobileNumberNotFound
		}
		// 其他特定错误会直接从 repo 传递上来
		return nil, err
	}
	return handledMobileNumber, nil
}
