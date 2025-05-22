package repositories

import (
	"context"

	"github.com/phone_management/internal/models"
	"gorm.io/gorm"
)

// VerificationTokenRepository 定义了验证令牌仓库的接口
type VerificationTokenRepository interface {
	Create(ctx context.Context, token *models.VerificationToken) error
	FindByToken(ctx context.Context, token string) (*models.VerificationToken, error)
	UpdateStatus(ctx context.Context, token string, status models.VerificationTokenStatus) error
	// Future methods for managing tokens can be added here, e.g., FindByToken, UpdateStatus
}

type gormVerificationTokenRepository struct {
	db *gorm.DB
}

// NewGormVerificationTokenRepository 创建一个新的 GORM 验证令牌仓库实例
func NewGormVerificationTokenRepository(db *gorm.DB) VerificationTokenRepository {
	return &gormVerificationTokenRepository{db: db}
}

// Create 在数据库中创建一个新的验证令牌记录
func (r *gormVerificationTokenRepository) Create(ctx context.Context, token *models.VerificationToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

// FindByToken 通过token查询验证令牌信息
func (r *gormVerificationTokenRepository) FindByToken(ctx context.Context, token string) (*models.VerificationToken, error) {
	var verificationToken models.VerificationToken
	err := r.db.WithContext(ctx).Where("token = ?", token).First(&verificationToken).Error
	if err != nil {
		return nil, err
	}
	return &verificationToken, nil
}

// UpdateStatus 更新验证令牌的状态
func (r *gormVerificationTokenRepository) UpdateStatus(ctx context.Context, token string, status models.VerificationTokenStatus) error {
	return r.db.WithContext(ctx).Model(&models.VerificationToken{}).Where("token = ?", token).Update("status", status).Error
}
