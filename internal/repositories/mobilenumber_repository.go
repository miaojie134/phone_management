package repositories

import (
	"errors"
	"strings"

	"github.com/phone_management/internal/models"
	"gorm.io/gorm"
)

// ErrPhoneNumberExists 表示手机号码已存在
var ErrPhoneNumberExists = errors.New("手机号码已存在")

// MobileNumberRepository 定义了手机号码数据仓库的接口
type MobileNumberRepository interface {
	CreateMobileNumber(mobileNumber *models.MobileNumber) (*models.MobileNumber, error)
	GetMobileNumbers(page, limit int, sortBy, sortOrder, search, status, applicantStatus string) ([]models.MobileNumberResponse, int64, error)
	//未来可以扩展其他方法，如 GetByPhoneNumber, Update, Delete 等
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
func (r *gormMobileNumberRepository) CreateMobileNumber(mobileNumber *models.MobileNumber) (*models.MobileNumber, error) {
	// GORM 在 Create 时会自动处理唯一约束错误。我们可以依赖它，或者预先检查。
	// 为了更明确的错误类型，可以预先检查：
	var existing models.MobileNumber
	if err := r.db.Where("phone_number = ?", mobileNumber.PhoneNumber).First(&existing).Error; err == nil {
		return nil, ErrPhoneNumberExists // 号码已存在
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err // 其他查询错误
	}

	// 如果记录未找到，则创建新记录
	if err := r.db.Create(mobileNumber).Error; err != nil {
		// GORM 通常会将数据库的唯一约束违例错误包装起来
		// 对于 SQLite，错误信息可能包含 "UNIQUE constraint failed"
		if strings.Contains(strings.ToLower(err.Error()), "unique constraint") || strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			return nil, ErrPhoneNumberExists
		}
		return nil, err
	}
	return mobileNumber, nil
}

// GetMobileNumbers 从数据库中获取手机号码列表，支持分页、排序、搜索和筛选
func (r *gormMobileNumberRepository) GetMobileNumbers(page, limit int, sortBy, sortOrder, search, status, applicantStatus string) ([]models.MobileNumberResponse, int64, error) {
	var mobileNumbers []models.MobileNumberResponse
	var totalItems int64

	tx := r.db.Model(&models.MobileNumber{}). // 开始构建查询，基于 MobileNumber 模型
							Select(
			"mobile_numbers.id AS id",
			"mobile_numbers.phone_number AS phone_number",
			"mobile_numbers.applicant_employee_db_id AS applicant_employee_db_id",
			"applicant.full_name AS applicant_name",           // 办卡人姓名
			"applicant.employment_status AS applicant_status", // 办卡人状态
			"mobile_numbers.application_date AS application_date",
			"mobile_numbers.current_employee_db_id AS current_employee_db_id",
			"current_user.full_name AS current_user_name", // 当前使用人姓名
			"mobile_numbers.status AS status",
			"mobile_numbers.vendor AS vendor",
			"mobile_numbers.remarks AS remarks",
			"mobile_numbers.cancellation_date AS cancellation_date",
			"mobile_numbers.created_at AS created_at",
			"mobile_numbers.updated_at AS updated_at",
		).
		Joins("LEFT JOIN employees AS applicant ON applicant.id = mobile_numbers.applicant_employee_db_id").
		Joins("LEFT JOIN employees AS current_user ON current_user.id = mobile_numbers.current_employee_db_id")

	// 处理搜索条件
	if search != "" {
		searchTerm := "%" + search + "%"
		tx = tx.Where(
			"mobile_numbers.phone_number LIKE ? OR applicant.full_name LIKE ? OR current_user.full_name LIKE ?",
			searchTerm, searchTerm, searchTerm,
		)
	}

	// 处理状态筛选
	if status != "" {
		tx = tx.Where("mobile_numbers.status = ?", status)
	}

	// 处理办卡人状态筛选
	if applicantStatus != "" {
		tx = tx.Where("applicant.employment_status = ?", applicantStatus)
	}

	// 计算总数（在应用分页之前）
	if err := tx.Count(&totalItems).Error; err != nil {
		return nil, 0, err
	}

	// 处理排序
	if sortBy != "" {
		// 白名单校验 sortBy 字段，防止 SQL 注入
		allowedSortByFields := map[string]string{
			"id":              "mobile_numbers.id",
			"phoneNumber":     "mobile_numbers.phone_number",
			"applicationDate": "mobile_numbers.application_date",
			"status":          "mobile_numbers.status",
			"vendor":          "mobile_numbers.vendor",
			"createdAt":       "mobile_numbers.created_at", // 默认排序字段之一
			"applicantName":   "applicant.full_name",
			"currentUserName": "current_user.full_name",
			"applicantStatus": "applicant.employment_status",
		}
		dbSortBy, isValidField := allowedSortByFields[sortBy]
		if !isValidField {
			dbSortBy = "mobile_numbers.created_at" // 如果字段无效，则使用默认排序字段
		}

		if strings.ToLower(sortOrder) != "desc" {
			sortOrder = "asc"
		}
		tx = tx.Order(dbSortBy + " " + sortOrder)
	} else {
		// 默认排序
		tx = tx.Order("mobile_numbers.created_at desc")
	}

	// 处理分页
	offset := (page - 1) * limit
	if err := tx.Offset(offset).Limit(limit).Find(&mobileNumbers).Error; err != nil {
		return nil, 0, err
	}

	return mobileNumbers, totalItems, nil
}
