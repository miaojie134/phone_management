package repositories

import (
	"context"

	"github.com/phone_management/internal/models"
	"gorm.io/gorm"
)

// VerificationTokenRepository 定义了验证令牌仓库的接口
type VerificationTokenRepository interface {
	Create(ctx context.Context, token *models.VerificationToken) error
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
