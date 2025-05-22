package repositories

import (
	"context"
	"strings"
	"time"

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
	// 以下是管理员查看状态API所需的方法
	CountReportedIssues(ctx context.Context) (int, error) // 统计所有报告的问题总数
	FindReportedIssuesWithDetails(ctx context.Context, employeeID, departmentName string) ([]models.ReportedIssueDetail, error)
	FindUnlistedNumbersWithDetails(ctx context.Context, employeeID, departmentName string) ([]models.ReportedUnlistedNumberInfo, error)
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

// CountReportedIssues 统计所有报告的问题总数
func (r *gormUserReportedIssueRepository) CountReportedIssues(ctx context.Context) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.UserReportedIssue{}).Count(&count).Error
	return int(count), err
}

// FindReportedIssuesWithDetails 查询用户报告的号码问题详情
func (r *gormUserReportedIssueRepository) FindReportedIssuesWithDetails(ctx context.Context, employeeID, departmentName string) ([]models.ReportedIssueDetail, error) {
	var results []struct {
		IssueID           uint      `gorm:"column:id"`
		PhoneNumber       string    `gorm:"column:phone_number"`
		ReportedBy        string    `gorm:"column:reported_by"`
		Comment           *string   `gorm:"column:user_comment"`
		OriginalStatus    string    `gorm:"column:status"`
		ReportedAt        time.Time `gorm:"column:created_at"`
		AdminActionStatus string    `gorm:"column:admin_action_status"`
	}

	query := r.db.WithContext(ctx).Table("user_reported_issues uri").
		Select("uri.id, mn.phone_number, e.full_name as reported_by, uri.user_comment, mn.status, uri.created_at, uri.admin_action_status").
		Joins("JOIN employees e ON uri.reported_by_employee_id = e.employee_id").
		Joins("JOIN mobile_numbers mn ON uri.mobile_number_db_id = mn.id").
		Where("uri.issue_type = ?", "number_issue")

	// 应用过滤条件
	if employeeID != "" {
		query = query.Where("uri.reported_by_employee_id = ?", employeeID)
	}
	if departmentName != "" {
		query = query.Where("e.department = ?", departmentName)
	}

	err := query.Order("uri.created_at desc").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// 转换为 ReportedIssueDetail 结构体
	reportedIssues := make([]models.ReportedIssueDetail, 0, len(results))
	for _, r := range results {
		var comment string
		if r.Comment != nil {
			comment = *r.Comment
		}

		// 解析 comment 中的 purpose 信息 (可能存在于 "用途应为: xxx" 格式中)
		var purpose *string
		if r.Comment != nil {
			// 简单处理，实际可能需要更复杂的逻辑
			if strings.Contains(*r.Comment, "用途应为:") || strings.Contains(*r.Comment, "报告用途:") {
				lines := strings.Split(*r.Comment, "\n")
				for _, line := range lines {
					if strings.Contains(line, "用途应为:") {
						p := strings.TrimSpace(strings.TrimPrefix(line, "用途应为:"))
						purpose = &p
						break
					}
					if strings.Contains(line, "报告用途:") {
						p := strings.TrimSpace(strings.TrimPrefix(line, "报告用途:"))
						purpose = &p
						break
					}
				}
			}
		}

		reportedIssues = append(reportedIssues, models.ReportedIssueDetail{
			IssueID:           r.IssueID,
			PhoneNumber:       r.PhoneNumber,
			ReportedBy:        r.ReportedBy,
			Comment:           comment,
			Purpose:           purpose,
			OriginalStatus:    r.OriginalStatus,
			ReportedAt:        r.ReportedAt,
			AdminActionStatus: r.AdminActionStatus,
		})
	}

	return reportedIssues, nil
}

// FindUnlistedNumbersWithDetails 查询用户报告的未列出号码详情
func (r *gormUserReportedIssueRepository) FindUnlistedNumbersWithDetails(ctx context.Context, employeeID, departmentName string) ([]models.ReportedUnlistedNumberInfo, error) {
	var results []struct {
		PhoneNumber *string   `gorm:"column:reported_phone_number"`
		ReportedBy  string    `gorm:"column:reported_by"`
		Comment     *string   `gorm:"column:user_comment"`
		ReportedAt  time.Time `gorm:"column:created_at"`
	}

	query := r.db.WithContext(ctx).Table("user_reported_issues uri").
		Select("uri.reported_phone_number, e.full_name as reported_by, uri.user_comment, uri.created_at").
		Joins("JOIN employees e ON uri.reported_by_employee_id = e.employee_id").
		Where("uri.issue_type = ?", "unlisted_number")

	// 应用过滤条件
	if employeeID != "" {
		query = query.Where("uri.reported_by_employee_id = ?", employeeID)
	}
	if departmentName != "" {
		query = query.Where("e.department = ?", departmentName)
	}

	err := query.Order("uri.created_at desc").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	// 转换为 ReportedUnlistedNumberInfo 结构体
	unlistedNumbers := make([]models.ReportedUnlistedNumberInfo, 0, len(results))
	for _, r := range results {
		if r.PhoneNumber != nil {
			var comment string
			if r.Comment != nil {
				comment = *r.Comment
			}

			unlistedNumbers = append(unlistedNumbers, models.ReportedUnlistedNumberInfo{
				PhoneNumber: *r.PhoneNumber,
				UserComment: comment,
				ReportedAt:  r.ReportedAt,
			})
		}
	}

	return unlistedNumbers, nil
}
