package repositories

import (
	"context"

	"github.com/phone_management/internal/models"
	"gorm.io/gorm"
)

// NumberApplicantHistoryRepository 定义了号码办卡人变更历史数据仓库的接口
type NumberApplicantHistoryRepository interface {
	// Create 创建办卡人变更历史记录
	Create(ctx context.Context, history *models.NumberApplicantHistory) error
	// GetByMobileNumberID 根据手机号码ID获取变更历史
	GetByMobileNumberID(ctx context.Context, mobileNumberID uint) ([]models.NumberApplicantHistory, error)
	// GetByApplicantID 根据员工ID获取其相关的变更历史
	GetByApplicantID(ctx context.Context, employeeID string) ([]models.NumberApplicantHistory, error)
}

// gormNumberApplicantHistoryRepository 是 NumberApplicantHistoryRepository 的 GORM 实现
type gormNumberApplicantHistoryRepository struct {
	db *gorm.DB
}

// NewGormNumberApplicantHistoryRepository 创建一个新的 gormNumberApplicantHistoryRepository 实例
func NewGormNumberApplicantHistoryRepository(db *gorm.DB) NumberApplicantHistoryRepository {
	return &gormNumberApplicantHistoryRepository{db: db}
}

// Create 创建办卡人变更历史记录
func (r *gormNumberApplicantHistoryRepository) Create(ctx context.Context, history *models.NumberApplicantHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

// GetByMobileNumberID 根据手机号码ID获取变更历史
func (r *gormNumberApplicantHistoryRepository) GetByMobileNumberID(ctx context.Context, mobileNumberID uint) ([]models.NumberApplicantHistory, error) {
	var histories []models.NumberApplicantHistory
	err := r.db.WithContext(ctx).
		Where("mobile_number_db_id = ?", mobileNumberID).
		Order("change_date DESC").
		Find(&histories).Error
	return histories, err
}

// GetByApplicantID 根据员工ID获取其相关的变更历史（作为原办卡人或新办卡人）
func (r *gormNumberApplicantHistoryRepository) GetByApplicantID(ctx context.Context, employeeID string) ([]models.NumberApplicantHistory, error) {
	var histories []models.NumberApplicantHistory
	err := r.db.WithContext(ctx).
		Where("previous_applicant_id = ? OR new_applicant_id = ?", employeeID, employeeID).
		Order("change_date DESC").
		Find(&histories).Error
	return histories, err
}
