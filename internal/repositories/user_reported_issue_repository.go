package repositories

import (
	"context"

	"github.com/phone_management/internal/models"
	"gorm.io/gorm"
)

// UserReportedIssueRepository 定义了用户报告问题仓库的接口
type UserReportedIssueRepository interface {
	// CreateReportedIssue 创建一个新的用户报告问题记录
	CreateReportedIssue(ctx context.Context, issue *models.UserReportedIssue) error
	// BatchCreateReportedIssues 批量创建用户报告问题记录
	BatchCreateReportedIssues(ctx context.Context, issues []*models.UserReportedIssue) error
	// FindReportedMobileNumberIdsByTokenId 查找特定验证令牌ID对应的已报告问题的手机号码ID列表
	FindReportedMobileNumberIdsByTokenId(ctx context.Context, tokenId uint) ([]uint, error)
	// FindUnlistedByTokenId 查找特定验证令牌ID对应的、类型为 unlisted_number 的报告问题记录
	FindUnlistedByTokenId(ctx context.Context, tokenId uint) ([]models.UserReportedIssue, error)
}

type gormUserReportedIssueRepository struct {
	db *gorm.DB
}

// NewGormUserReportedIssueRepository 创建一个新的GORM用户报告问题仓库实例
func NewGormUserReportedIssueRepository(db *gorm.DB) UserReportedIssueRepository {
	return &gormUserReportedIssueRepository{db: db}
}

// CreateReportedIssue 创建一个新的用户报告问题记录
func (r *gormUserReportedIssueRepository) CreateReportedIssue(ctx context.Context, issue *models.UserReportedIssue) error {
	return r.db.WithContext(ctx).Create(issue).Error
}

// BatchCreateReportedIssues 批量创建用户报告问题记录
func (r *gormUserReportedIssueRepository) BatchCreateReportedIssues(ctx context.Context, issues []*models.UserReportedIssue) error {
	return r.db.WithContext(ctx).Create(issues).Error
}

// FindReportedMobileNumberIdsByTokenId 查找特定验证令牌ID对应的已报告问题的手机号码ID列表
func (r *gormUserReportedIssueRepository) FindReportedMobileNumberIdsByTokenId(ctx context.Context, tokenId uint) ([]uint, error) {
	var results []struct {
		MobileNumberDbId uint
	}

	err := r.db.WithContext(ctx).Model(&models.UserReportedIssue{}).
		Where("verification_token_id = ? AND mobile_number_db_id IS NOT NULL", tokenId).
		Select("mobile_number_db_id").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// 提取ID列表
	ids := make([]uint, 0, len(results))
	for _, result := range results {
		ids = append(ids, result.MobileNumberDbId)
	}

	return ids, nil
}

// FindUnlistedByTokenId 查找特定验证令牌ID对应的、类型为 unlisted_number 的报告问题记录
func (r *gormUserReportedIssueRepository) FindUnlistedByTokenId(ctx context.Context, tokenId uint) ([]models.UserReportedIssue, error) {
	var issues []models.UserReportedIssue
	err := r.db.WithContext(ctx).
		Where("verification_token_id = ? AND issue_type = ?", tokenId, "unlisted_number").
		Order("created_at desc"). // 按创建时间降序排列
		Find(&issues).Error
	return issues, err
}
