package repositories

import (
	"errors"
	"strings"
	"time"

	"github.com/phone_management/internal/models"
	"gorm.io/gorm"
)

// ErrPhoneNumberExists 表示手机号码已存在
var ErrPhoneNumberExists = errors.New("手机号码已存在")

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
	CreateMobileNumber(mobileNumber *models.MobileNumber) (*models.MobileNumber, error)
	GetMobileNumbers(page, limit int, sortBy, sortOrder, search, status, applicantStatus string) ([]models.MobileNumberResponse, int64, error)
	GetMobileNumberByID(id uint) (*models.MobileNumberResponse, error)
	//未来可以扩展其他方法，如 GetByPhoneNumber, Update, Delete 等
	UpdateMobileNumber(id uint, updates map[string]interface{}) (*models.MobileNumber, error)
	AssignMobileNumber(numberID uint, employeeID uint, assignmentDate time.Time) (*models.MobileNumber, error)
	UnassignMobileNumber(numberID uint, reclaimDate time.Time) (*models.MobileNumber, error)
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

// GetMobileNumberByID 从数据库中获取指定ID的手机号码详情及其使用历史
func (r *gormMobileNumberRepository) GetMobileNumberByID(id uint) (*models.MobileNumberResponse, error) {
	var mobileNumberDetail models.MobileNumberResponse

	// 1. 获取手机号码基本信息并关联办卡人和当前使用人姓名及办卡人状态
	tx := r.db.Model(&models.MobileNumber{}).
		Select(
			"mobile_numbers.id AS id",
			"mobile_numbers.phone_number AS phone_number",
			"mobile_numbers.applicant_employee_db_id AS applicant_employee_db_id",
			"applicant.full_name AS applicant_name",
			"applicant.employment_status AS applicant_status",
			"mobile_numbers.application_date AS application_date",
			"mobile_numbers.current_employee_db_id AS current_employee_db_id",
			"current_user.full_name AS current_user_name",
			"mobile_numbers.status AS status",
			"mobile_numbers.vendor AS vendor",
			"mobile_numbers.remarks AS remarks",
			"mobile_numbers.cancellation_date AS cancellation_date",
			"mobile_numbers.created_at AS created_at",
			"mobile_numbers.updated_at AS updated_at",
		).
		Joins("LEFT JOIN employees AS applicant ON applicant.id = mobile_numbers.applicant_employee_db_id").
		Joins("LEFT JOIN employees AS current_user ON current_user.id = mobile_numbers.current_employee_db_id").
		Where("mobile_numbers.id = ?", id).
		First(&mobileNumberDetail)

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound // 返回仓库层定义的 ErrRecordNotFound
		}
		return nil, tx.Error
	}

	// 2. 获取该号码的使用历史
	var usageHistory []models.NumberUsageHistory
	if err := r.db.Where("mobile_number_db_id = ?", id).Order("start_date desc").Find(&usageHistory).Error; err != nil {
		// 如果获取使用历史失败，可以根据业务决定是返回错误还是仅记录日志并返回部分数据
		// 这里选择返回错误
		return nil, err
	}
	mobileNumberDetail.UsageHistory = usageHistory

	return &mobileNumberDetail, nil
}

// UpdateMobileNumber 更新数据库中的手机号码记录
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

// AssignMobileNumber 将手机号码分配给员工
func (r *gormMobileNumberRepository) AssignMobileNumber(numberID uint, employeeID uint, assignmentDate time.Time) (*models.MobileNumber, error) {
	var mobileNumber models.MobileNumber
	var employee models.Employee

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 1. 查找并锁定手机号码记录
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&mobileNumber, numberID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRecordNotFound // 使用仓库层已定义的错误
			}
			return err
		}

		// 2. 校验号码是否为"闲置"状态
		if mobileNumber.Status != string(models.StatusIdle) {
			return ErrMobileNumberNotInIdleStatus
		}

		// 3. 查找员工记录
		if err := tx.First(&employee, employeeID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEmployeeNotFound
			}
			return err
		}

		// 4. 校验员工是否为"在职"状态
		// 假设员工模型中 EmploymentStatus 'Active' 表示在职
		if employee.EmploymentStatus != "Active" { // TODO: Consider using a constant for "Active" status if defined elsewhere
			return ErrEmployeeNotActive
		}

		// 5. 更新号码记录
		mobileNumber.CurrentEmployeeDbID = &employeeID
		mobileNumber.Status = string(models.StatusInUse)
		if err := tx.Save(&mobileNumber).Error; err != nil {
			return err
		}

		// 6. 创建一条新的号码使用历史记录
		usageHistory := models.NumberUsageHistory{
			MobileNumberDbID: int64(numberID), // GORM 会自动处理类型转换，但明确类型更好
			EmployeeDbID:     int64(employeeID),
			StartDate:        assignmentDate,
		}
		if err := tx.Create(&usageHistory).Error; err != nil {
			return err
		}

		return nil // 事务成功
	})

	if err != nil {
		return nil, err // 返回事务中发生的任何错误
	}

	// 成功后，返回更新后的手机号码信息（可以重新查询以获取最新关联数据，但此处直接返回事务中修改的对象）
	// 为了确保返回的数据是最新的，尤其是如果 NumberUsageHistory 需要被嵌入，最好重新查询
	// 但基于 API 文档，返回更新后的号码对象即可，当前 mobileNumber 对象已更新。
	// 如果需要返回包含办卡人姓名等，则需要重新调用 GetMobileNumberByID 或类似方法。
	// 但 Assign 操作本身返回的是 MobileNumber 模型。
	return &mobileNumber, nil
}

// UnassignMobileNumber 从当前使用人处回收手机号码
func (r *gormMobileNumberRepository) UnassignMobileNumber(numberID uint, reclaimDate time.Time) (*models.MobileNumber, error) {
	var mobileNumber models.MobileNumber

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 1. 查找并锁定手机号码记录
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&mobileNumber, numberID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRecordNotFound // 号码未找到
			}
			return err
		}

		// 2. 校验号码是否为"在用"状态
		if mobileNumber.Status != string(models.StatusInUse) {
			return ErrMobileNumberNotInUseStatus
		}

		// 3. 更新号码使用历史记录：找到当前这条号码的最后一条（即当前生效的）使用记录，并设置其 endDate
		var usageHistory models.NumberUsageHistory
		// 我们需要找到该号码 (MobileNumberDbID) 当前使用者 (EmployeeDbID) 的那条未结束的记录
		// mobileNumber.CurrentEmployeeDbID 应该是有值的，因为状态是在用
		if mobileNumber.CurrentEmployeeDbID == nil {
			// 这是一个数据不一致的情况，理论上"在用"的号码应该有关联的 currentEmployeeDbId
			return errors.New("数据不一致：在用号码没有关联当前用户")
		}

		result := tx.Where("mobile_number_db_id = ? AND employee_db_id = ? AND end_date IS NULL",
			numberID, *mobileNumber.CurrentEmployeeDbID).
			Order("start_date desc"). // 理论上每个用户对一个号码同时只有一条active记录，但以防万一取最新的
			First(&usageHistory)

		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// 如果没有找到匹配的 usage history，这可能意味着数据存在问题，或者逻辑需要调整
				// 例如，号码被标记为"在用"，但没有对应的"未结束"的使用记录
				return ErrNoActiveUsageHistoryFound
			}
			return result.Error
		}

		usageHistory.EndDate = &reclaimDate
		if err := tx.Save(&usageHistory).Error; err != nil {
			return err
		}

		// 4. 更新号码记录：清空当前使用人，状态改回"闲置"
		mobileNumber.CurrentEmployeeDbID = nil
		mobileNumber.Status = string(models.StatusIdle)
		if err := tx.Save(&mobileNumber).Error; err != nil {
			return err
		}

		return nil // 事务成功
	})

	if err != nil {
		return nil, err
	}

	return &mobileNumber, nil
}
