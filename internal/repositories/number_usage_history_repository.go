package repositories

import (
	"context"
	"time"

	"github.com/phone_management/internal/models"
	"gorm.io/gorm"
)

// NumberUsageHistoryRepository 定义了号码使用历史数据仓库的接口
type NumberUsageHistoryRepository interface {
	// Create 创建号码使用历史记录
	Create(ctx context.Context, history *models.NumberUsageHistory) error
	// GetByMobileNumberID 根据手机号码ID获取使用历史
	GetByMobileNumberID(ctx context.Context, mobileNumberID uint) ([]models.NumberUsageHistory, error)
	// GetByEmployeeID 根据员工ID获取其使用历史
	GetByEmployeeID(ctx context.Context, employeeID string) ([]models.NumberUsageHistory, error)
	// GetActiveUsageByMobileNumberAndEmployee 获取指定号码和员工的当前有效使用记录
	GetActiveUsageByMobileNumberAndEmployee(ctx context.Context, mobileNumberID uint, employeeID string) (*models.NumberUsageHistory, error)
	// UpdateEndDate 更新使用历史的结束时间
	UpdateEndDate(ctx context.Context, historyID int64, endDate time.Time) error
}

// gormNumberUsageHistoryRepository 是 NumberUsageHistoryRepository 的 GORM 实现
type gormNumberUsageHistoryRepository struct {
	db *gorm.DB
}

// NewGormNumberUsageHistoryRepository 创建一个新的 gormNumberUsageHistoryRepository 实例
func NewGormNumberUsageHistoryRepository(db *gorm.DB) NumberUsageHistoryRepository {
	return &gormNumberUsageHistoryRepository{db: db}
}

// Create 创建号码使用历史记录
func (r *gormNumberUsageHistoryRepository) Create(ctx context.Context, history *models.NumberUsageHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

// GetByMobileNumberID 根据手机号码ID获取使用历史
func (r *gormNumberUsageHistoryRepository) GetByMobileNumberID(ctx context.Context, mobileNumberID uint) ([]models.NumberUsageHistory, error) {
	var histories []models.NumberUsageHistory
	err := r.db.WithContext(ctx).
		Where("mobile_number_db_id = ?", mobileNumberID).
		Order("start_date DESC").
		Find(&histories).Error
	return histories, err
}

// GetByEmployeeID 根据员工ID获取其使用历史
func (r *gormNumberUsageHistoryRepository) GetByEmployeeID(ctx context.Context, employeeID string) ([]models.NumberUsageHistory, error) {
	var histories []models.NumberUsageHistory
	err := r.db.WithContext(ctx).
		Where("employee_id = ?", employeeID).
		Order("start_date DESC").
		Find(&histories).Error
	return histories, err
}

// GetActiveUsageByMobileNumberAndEmployee 获取指定号码和员工的当前有效使用记录（未结束的记录）
func (r *gormNumberUsageHistoryRepository) GetActiveUsageByMobileNumberAndEmployee(ctx context.Context, mobileNumberID uint, employeeID string) (*models.NumberUsageHistory, error) {
	var history models.NumberUsageHistory
	err := r.db.WithContext(ctx).
		Where("mobile_number_db_id = ? AND employee_id = ? AND end_date IS NULL", mobileNumberID, employeeID).
		Order("start_date DESC").
		First(&history).Error

	if err != nil {
		return nil, err
	}
	return &history, nil
}

// UpdateEndDate 更新使用历史的结束时间
func (r *gormNumberUsageHistoryRepository) UpdateEndDate(ctx context.Context, historyID int64, endDate time.Time) error {
	return r.db.WithContext(ctx).
		Model(&models.NumberUsageHistory{}).
		Where("id = ?", historyID).
		Update("end_date", endDate).Error
}
