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
