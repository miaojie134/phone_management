package repositories

import (
	"context"
	"time"

	"github.com/phone_management/internal/models"
	"gorm.io/gorm"
)

// VerificationTokenRepository 定义了验证令牌仓库的接口
type VerificationTokenRepository interface {
	Create(ctx context.Context, token *models.VerificationToken) error
	FindByToken(ctx context.Context, token string) (*models.VerificationToken, error)
	UpdateStatus(ctx context.Context, token string, status models.VerificationTokenStatus) error
	FindPendingTokensWithEmployeeInfo(ctx context.Context, employeeID, departmentName string) ([]models.PendingUserDetail, error)
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

// FindPendingTokensWithEmployeeInfo 查询未响应的令牌及相关员工信息
func (r *gormVerificationTokenRepository) FindPendingTokensWithEmployeeInfo(ctx context.Context, employeeID, departmentName string) ([]models.PendingUserDetail, error) {
	var results []struct {
		TokenID    uint      `gorm:"column:id"`
		EmployeeID string    `gorm:"column:employee_id"`
		ExpiresAt  time.Time `gorm:"column:expires_at"`
		FullName   string    `gorm:"column:full_name"`
		Email      *string   `gorm:"column:email"`
		Department *string   `gorm:"column:department"`
	}

	query := r.db.WithContext(ctx).Table("verification_tokens vt").
		Select("vt.id, vt.employee_id, vt.expires_at, e.full_name, e.email, e.department").
		Joins("JOIN employees e ON vt.employee_id = e.employee_id").
		Where("vt.status = ? AND vt.expires_at > ?", models.VerificationTokenStatusPending, time.Now())

	// 应用过滤条件
	if employeeID != "" {
		query = query.Where("vt.employee_id = ?", employeeID)
	}
	if departmentName != "" {
		query = query.Where("e.department = ?", departmentName)
	}

	err := query.Order("vt.expires_at asc").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// 转换为 PendingUserDetail 结构体
	pendingUsers := make([]models.PendingUserDetail, 0, len(results))
	for _, r := range results {
		expiresAt := r.ExpiresAt
		pendingUsers = append(pendingUsers, models.PendingUserDetail{
			EmployeeID: r.EmployeeID,
			FullName:   r.FullName,
			Email:      r.Email,
			TokenID:    r.TokenID,
			ExpiresAt:  &expiresAt,
		})
	}

	return pendingUsers, nil
}
