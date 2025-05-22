package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/phone_management/internal/models"
	"gorm.io/gorm"
)

// ErrMobileNumberStringConflict 表示该手机号码的记录已存在
var ErrMobileNumberStringConflict = errors.New("该手机号码的记录已存在")

// ErrRecordNotFound 表示记录未找到，可以重用 gorm 的错误或自定义
var ErrRecordNotFound = gorm.ErrRecordNotFound

// New errors for assign operation
var ErrMobileNumberNotInIdleStatus = errors.New("手机号码不是闲置状态")
var ErrEmployeeNotFound = errors.New("员工未找到")
var ErrEmployeeNotActive = errors.New("员工不是在职状态")
var ErrMobileNumberNotInUseStatus = errors.New("手机号码不是在用状态")
var ErrNoActiveUsageHistoryFound = errors.New("未找到该号码当前有效的分配记录")

// MobileNumberRepository 定义了手机号码数据仓库的接口
type MobileNumberRepository interface {
	// CreateMobileNumber 的第二个参数 mobileNumber 中已包含 ApplicantEmployeeID (string)
	CreateMobileNumber(mobileNumber *models.MobileNumber) (*models.MobileNumber, error)
	GetMobileNumbers(page, limit int, sortBy, sortOrder, search, status, applicantStatus string) ([]models.MobileNumberResponse, int64, error)
	GetMobileNumberResponseByPhoneNumber(phoneNumber string) (*models.MobileNumberResponse, error)
	GetMobileNumberByPhoneNumber(phoneNumber string) (*models.MobileNumber, error)
	//未来可以扩展其他方法，如 GetByPhoneNumber, Update, Delete 等
	UpdateMobileNumber(id uint, updates map[string]interface{}) (*models.MobileNumber, error)
	// AssignMobileNumber 的第二个参数 employeeBusinessID 应该是 string (业务工号)
	AssignMobileNumber(numberID uint, employeeBusinessID string, assignmentDate time.Time) (*models.MobileNumber, error)
	UnassignMobileNumber(numberID uint, reclaimDate time.Time) (*models.MobileNumber, error)
	// FindAssignedToEmployee 查询分配给特定员工的手机号码
	FindAssignedToEmployee(ctx context.Context, employeeID string) ([]models.MobileNumber, error)
	// UpdateLastConfirmationDate 更新号码的最后确认日期
	UpdateLastConfirmationDate(ctx context.Context, numberID uint) error
	// MarkAsReportedByUser 将号码标记为用户报告问题
	MarkAsReportedByUser(ctx context.Context, numberID uint) error
}

// gormMobileNumberRepository 是 MobileNumberRepository 的 GORM 实现
type gormMobileNumberRepository struct {
	db *gorm.DB
}

// NewGormMobileNumberRepository 创建一个新的 gormMobileNumberRepository 实例
func NewGormMobileNumberRepository(db *gorm.DB) MobileNumberRepository {
	return &gormMobileNumberRepository{db: db}
}

// CreateMobileNumber 在数据库中创建一个新的手机号码记录
// mobileNumber.ApplicantEmployeeID (string) 已经在模型中设置
func (r *gormMobileNumberRepository) CreateMobileNumber(mobileNumber *models.MobileNumber) (*models.MobileNumber, error) {
	var existing models.MobileNumber
	// 检查 phone_number 字符串是否已作为记录存在
	if err := r.db.Where("phone_number = ?", mobileNumber.PhoneNumber).First(&existing).Error; err == nil {
		return nil, ErrMobileNumberStringConflict // 使用新的错误变量名
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err := r.db.Create(mobileNumber).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique constraint") || strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			if strings.Contains(err.Error(), models.MobileNumber{}.TableName()+".phone_number") || strings.Contains(err.Error(), "MobileNumbers.phone_number") {
				return nil, ErrMobileNumberStringConflict // 使用新的错误变量名
			}
		}
		return nil, err
	}
	return mobileNumber, nil
}

// GetMobileNumbers 从数据库中获取手机号码列表，支持分页、排序、搜索和筛选
// func (r *gormMobileNumberRepository) GetMobileNumbers(page, limit int, sortBy, sortOrder, search, status, applicantStatus string) ([]models.MobileNumberResponse, int64, error) {
// 	var mobileNumbers []models.MobileNumberResponse
// 	var totalItems int64

// 	tx := r.db.Model(&models.MobileNumber{}).
// 		Select(
// 			"mobile_numbers.id AS id",
// 			"mobile_numbers.phone_number AS phone_number",
// 			"mobile_numbers.applicant_employee_id AS applicant_employee_id", // 使用新的列名
// 			"applicant.full_name AS applicant_name",
// 			"applicant.employment_status AS applicant_status",
// 			"mobile_numbers.application_date AS application_date",
// 			"mobile_numbers.current_employee_id AS current_employee_id", // 使用新的列名
// 			"current_user.full_name AS current_user_name",
// 			"mobile_numbers.status AS status",
// 			"mobile_numbers.vendor AS vendor",
// 			"mobile_numbers.remarks AS remarks",
// 			"mobile_numbers.cancellation_date AS cancellation_date",
// 			"mobile_numbers.created_at AS created_at",
// 			"mobile_numbers.updated_at AS updated_at",
// 		).
// 		Joins("LEFT JOIN employees AS applicant ON applicant.employee_id = mobile_numbers.applicant_employee_id").     // 连接条件改为业务工号
// 		Joins("LEFT JOIN employees AS current_user ON current_user.employee_id = mobile_numbers.current_employee_id"). // 连接条件改为业务工号
// 		Where("mobile_numbers.status = ?", status)

// 	// 处理搜索条件
// 	if search != "" {
// 		searchTerm := "%" + search + "%"
// 		tx = tx.Where(
// 			"mobile_numbers.phone_number LIKE ? OR applicant.full_name LIKE ? OR current_user.full_name LIKE ?",
// 			searchTerm, searchTerm, searchTerm,
// 		)
// 	}

// 	// 处理办卡人状态筛选
// 	if applicantStatus != "" {
// 		tx = tx.Where("applicant.employment_status = ?", applicantStatus)
// 	}

// 	// 计算总数（在应用分页之前）
// 	if err := tx.Count(&totalItems).Error; err != nil {
// 		return nil, 0, err
// 	}

// 	// 处理排序
// 	if sortBy != "" {
// 		// 白名单校验 sortBy 字段，防止 SQL 注入
// 		allowedSortByFields := map[string]string{
// 			"id":              "mobile_numbers.id",
// 			"phoneNumber":     "mobile_numbers.phone_number",
// 			"applicationDate": "mobile_numbers.application_date",
// 			"status":          "mobile_numbers.status",
// 			"vendor":          "mobile_numbers.vendor",
// 			"createdAt":       "mobile_numbers.created_at", // 默认排序字段之一
// 			"applicantName":   "applicant.full_name",
// 			"currentUserName": "current_user.full_name",
// 			"applicantStatus": "applicant.employment_status",
// 		}
// 		dbSortBy, isValidField := allowedSortByFields[sortBy]
// 		if !isValidField {
// 			dbSortBy = "mobile_numbers.created_at" // 如果字段无效，则使用默认排序字段
// 		}

// 		if strings.ToLower(sortOrder) != "desc" {
// 			sortOrder = "asc"
// 		}
// 		tx = tx.Order(dbSortBy + " " + sortOrder)
// 	} else {
// 		// 默认排序
// 		tx = tx.Order("mobile_numbers.created_at desc")
// 	}

// 	// 处理分页
// 	offset := (page - 1) * limit
// 	if err := tx.Offset(offset).Limit(limit).Find(&mobileNumbers).Error; err != nil {
// 		return nil, 0, err
// 	}

// 	return mobileNumbers, totalItems, nil
// }

// GetMobileNumbers 从数据库中获取手机号码列表，支持分页、排序、搜索和筛选
func (r *gormMobileNumberRepository) GetMobileNumbers(page, limit int, sortBy, sortOrder, search, status, applicantStatus string) ([]models.MobileNumberResponse, int64, error) {
	var mobileNumbers []models.MobileNumberResponse
	var totalItems int64

	// 基础查询构建器 (不包含 SELECT specific to response, or ORDER/LIMIT/OFFSET yet)
	queryBuilder := r.db.Model(&models.MobileNumber{}).
		Joins("LEFT JOIN employees AS applicant ON applicant.employee_id = mobile_numbers.applicant_employee_id").
		Joins("LEFT JOIN employees AS current_user ON current_user.employee_id = mobile_numbers.current_employee_id")

	// 应用可选的过滤条件
	if search != "" {
		searchTerm := "%" + search + "%"
		queryBuilder = queryBuilder.Where("mobile_numbers.phone_number LIKE ? OR applicant.full_name LIKE ? OR current_user.full_name LIKE ?", searchTerm, searchTerm, searchTerm)
	}
	if status != "" { // 仅当 status 参数非空时才应用此条件
		queryBuilder = queryBuilder.Where("mobile_numbers.status = ?", status)
	}
	if applicantStatus != "" { // 仅当 applicantStatus 参数非空时才应用此条件
		queryBuilder = queryBuilder.Where("applicant.employment_status = ?", applicantStatus)
	}

	// 执行 COUNT 查询获取总数 (基于已应用的过滤器)
	if err := queryBuilder.Count(&totalItems).Error; err != nil {
		return nil, 0, err
	}

	// 为 SELECT 查询准备字段
	selectFields := []string{
		"mobile_numbers.id AS id",
		"mobile_numbers.phone_number AS phone_number",
		"mobile_numbers.applicant_employee_id AS applicant_employee_id",
		"applicant.full_name AS applicant_name",
		"applicant.employment_status AS applicant_status",
		"mobile_numbers.application_date AS application_date",
		"mobile_numbers.current_employee_id AS current_employee_id",
		"current_user.full_name AS current_user_name",
		"mobile_numbers.status AS status",
		"mobile_numbers.vendor AS vendor",
		"mobile_numbers.remarks AS remarks",
		"mobile_numbers.cancellation_date AS cancellation_date",
		"mobile_numbers.created_at AS created_at",
		"mobile_numbers.updated_at AS updated_at",
	}

	// 应用 SELECT, ORDER BY, OFFSET, LIMIT 到查询构建器
	queryBuilder = queryBuilder.Select(selectFields)

	// 处理排序
	if sortBy != "" {
		allowedSortByFields := map[string]string{
			"id":              "mobile_numbers.id",
			"phoneNumber":     "mobile_numbers.phone_number",
			"applicationDate": "mobile_numbers.application_date",
			"status":          "mobile_numbers.status",
			"vendor":          "mobile_numbers.vendor",
			"createdAt":       "mobile_numbers.created_at",
			"applicantName":   "applicant.full_name",
			"currentUserName": "current_user.full_name",
			"applicantStatus": "applicant.employment_status",
		}
		dbSortBy, isValidField := allowedSortByFields[sortBy]
		if !isValidField {
			dbSortBy = "mobile_numbers.created_at" // 默认排序字段
		}
		if strings.ToLower(sortOrder) != "desc" {
			sortOrder = "asc"
		}
		queryBuilder = queryBuilder.Order(dbSortBy + " " + sortOrder)
	} else {
		// 默认排序
		queryBuilder = queryBuilder.Order("mobile_numbers.created_at desc")
	}

	// 处理分页
	offset := (page - 1) * limit
	queryBuilder = queryBuilder.Offset(offset).Limit(limit)

	// 执行最终查询获取数据列表
	if err := queryBuilder.Find(&mobileNumbers).Error; err != nil {
		return nil, 0, err
	}

	return mobileNumbers, totalItems, nil
}

// GetMobileNumberResponseByPhoneNumber 根据手机号码字符串查询手机号码详细信息，包括关联数据
func (r *gormMobileNumberRepository) GetMobileNumberResponseByPhoneNumber(phoneNumber string) (*models.MobileNumberResponse, error) {
	var mobileNumberDetail models.MobileNumberResponse

	tx := r.db.Model(&models.MobileNumber{}).
		Select(
			"mobile_numbers.id AS id",
			"mobile_numbers.phone_number AS phone_number",
			"mobile_numbers.applicant_employee_id AS applicant_employee_id",
			"applicant.full_name AS applicant_name",
			"applicant.employment_status AS applicant_status",
			"mobile_numbers.application_date AS application_date",
			"mobile_numbers.current_employee_id AS current_employee_id",
			"current_user.full_name AS current_user_name",
			"mobile_numbers.status AS status",
			"mobile_numbers.vendor AS vendor",
			"mobile_numbers.remarks AS remarks",
			"mobile_numbers.cancellation_date AS cancellation_date",
			"mobile_numbers.created_at AS created_at",
			"mobile_numbers.updated_at AS updated_at",
		).
		Joins("LEFT JOIN employees AS applicant ON applicant.employee_id = mobile_numbers.applicant_employee_id").
		Joins("LEFT JOIN employees AS current_user ON current_user.employee_id = mobile_numbers.current_employee_id").
		Where("mobile_numbers.phone_number = ?", phoneNumber). // 查询条件改为 phone_number
		First(&mobileNumberDetail)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, tx.Error
	}

	// 加载使用历史
	if err := r.db.Model(&models.NumberUsageHistory{}).Where("mobile_number_db_id = ?", mobileNumberDetail.ID).Order("start_date desc").Find(&mobileNumberDetail.UsageHistory).Error; err != nil {
		return nil, err
	}

	return &mobileNumberDetail, nil
}

// GetMobileNumberByPhoneNumber 根据手机号码字符串查询手机号码信息
// 注意：这里返回 *models.MobileNumber 而不是 *models.MobileNumberResponse
// 因为服务层可能需要原始模型对象进行操作，例如获取其数据库ID
func (r *gormMobileNumberRepository) GetMobileNumberByPhoneNumber(phoneNumber string) (*models.MobileNumber, error) {
	var mobileNumber models.MobileNumber
	if err := r.db.Where("phone_number = ?", phoneNumber).First(&mobileNumber).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound // 使用仓库已定义的错误
		}
		return nil, err
	}
	return &mobileNumber, nil
}

// UpdateMobileNumber 更新数据库中指定ID的手机号码信息
// updates 是一个包含要更新字段及其新值的 map
func (r *gormMobileNumberRepository) UpdateMobileNumber(id uint, updates map[string]interface{}) (*models.MobileNumber, error) {
	var mobileNumber models.MobileNumber
	// 首先，检查记录是否存在
	if err := r.db.First(&mobileNumber, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	// 更新记录
	// 使用 Model(&models.MobileNumber{}) 指定模型，并通过 Where 更新特定ID的记录
	if err := r.db.Model(&models.MobileNumber{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 重新查询更新后的记录并返回
	if err := r.db.First(&mobileNumber, id).Error; err != nil {
		return nil, err // 理论上此时应该能找到
	}

	return &mobileNumber, nil
}

// AssignMobileNumber 将手机号码分配给员工 (employeeID 为业务工号)
func (r *gormMobileNumberRepository) AssignMobileNumber(numberID uint, employeeBusinessID string, assignmentDate time.Time) (*models.MobileNumber, error) {
	var mobileNumber models.MobileNumber
	var employee models.Employee // 用于校验员工状态

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&mobileNumber, numberID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRecordNotFound
			}
			return err
		}

		if mobileNumber.Status != string(models.StatusIdle) {
			return ErrMobileNumberNotInIdleStatus
		}

		// 查找员工记录以校验状态 (使用业务工号)
		if err := tx.Where("employee_id = ?", employeeBusinessID).First(&employee).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEmployeeNotFound // 员工业务工号未找到
			}
			return err
		}

		if employee.EmploymentStatus != "Active" {
			return ErrEmployeeNotActive
		}

		mobileNumber.CurrentEmployeeID = &employeeBusinessID // 直接存储业务工号
		mobileNumber.Status = string(models.StatusInUse)
		if err := tx.Save(&mobileNumber).Error; err != nil {
			return err
		}

		usageHistory := models.NumberUsageHistory{
			MobileNumberDbID: int64(numberID),
			EmployeeID:       employeeBusinessID, // 存储业务工号
			StartDate:        assignmentDate,
		}
		if err := tx.Create(&usageHistory).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &mobileNumber, nil
}

// UnassignMobileNumber 从当前使用人处回收手机号码
func (r *gormMobileNumberRepository) UnassignMobileNumber(numberID uint, reclaimDate time.Time) (*models.MobileNumber, error) {
	var mobileNumber models.MobileNumber

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&mobileNumber, numberID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRecordNotFound
			}
			return err
		}

		if mobileNumber.Status != string(models.StatusInUse) {
			return ErrMobileNumberNotInUseStatus
		}

		if mobileNumber.CurrentEmployeeID == nil || *mobileNumber.CurrentEmployeeID == "" {
			return errors.New("数据不一致：在用号码没有关联当前用户业务工号")
		}

		var usageHistory models.NumberUsageHistory
		result := tx.Where("mobile_number_db_id = ? AND employee_id = ? AND end_date IS NULL",
			numberID, *mobileNumber.CurrentEmployeeID). // 使用业务工号查询
			Order("start_date desc").
			First(&usageHistory)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return ErrNoActiveUsageHistoryFound
			}
			return result.Error
		}

		usageHistory.EndDate = &reclaimDate
		if err := tx.Save(&usageHistory).Error; err != nil {
			return err
		}

		mobileNumber.CurrentEmployeeID = nil // 清空业务工号
		mobileNumber.Status = string(models.StatusIdle)
		if err := tx.Save(&mobileNumber).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &mobileNumber, nil
}

// FindAssignedToEmployee 查询分配给特定员工的手机号码
func (r *gormMobileNumberRepository) FindAssignedToEmployee(ctx context.Context, employeeID string) ([]models.MobileNumber, error) {
	var numbers []models.MobileNumber
	err := r.db.WithContext(ctx).Where("current_employee_id = ?", employeeID).Find(&numbers).Error
	if err != nil {
		return nil, err
	}
	return numbers, nil
}

// UpdateLastConfirmationDate 更新号码的最后确认日期
func (r *gormMobileNumberRepository) UpdateLastConfirmationDate(ctx context.Context, numberID uint) error {
	return r.db.WithContext(ctx).Model(&models.MobileNumber{}).
		Where("id = ?", numberID).
		Update("last_confirmation_date", time.Now()).
		Error
}

// MarkAsReportedByUser 将号码标记为用户报告问题
func (r *gormMobileNumberRepository) MarkAsReportedByUser(ctx context.Context, numberID uint) error {
	return r.db.WithContext(ctx).Model(&models.MobileNumber{}).
		Where("id = ?", numberID).
		Update("status", "待核实-用户报告").
		Error
}
