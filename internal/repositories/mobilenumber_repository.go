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
