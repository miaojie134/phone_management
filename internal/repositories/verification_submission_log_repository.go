package repositories

import (
	"context"
	"time"

	"github.com/phone_management/internal/models"
	"gorm.io/gorm"
)

// VerificationSubmissionLogRepository 定义了验证提交日志仓库的接口
type VerificationSubmissionLogRepository interface {
	// 创建日志记录
	Create(ctx context.Context, log *models.VerificationSubmissionLog) error
	// 批量创建日志记录
	BatchCreate(ctx context.Context, logs []*models.VerificationSubmissionLog) error

	// 统计不同操作类型的手机号码数量
	CountTotalPhones(ctx context.Context) (int, error)
	CountConfirmedPhones(ctx context.Context) (int, error)
	CountReportedIssuePhones(ctx context.Context) (int, error)
	CountPendingPhones(ctx context.Context) (int, error)
	CountNewlyReportedPhones(ctx context.Context) (int, error)

	// 查询最新的验证操作记录
	FindLatestActionsByPhoneNumber(ctx context.Context) (map[string]models.VerificationActionType, error)

	// 查询已确认使用的手机号码详情
	FindConfirmedPhoneDetails(ctx context.Context) ([]models.ConfirmedPhoneDetail, error)
}

type gormVerificationSubmissionLogRepository struct {
	db *gorm.DB
}

// NewGormVerificationSubmissionLogRepository 创建一个新的GORM验证提交日志仓库实例
func NewGormVerificationSubmissionLogRepository(db *gorm.DB) VerificationSubmissionLogRepository {
	return &gormVerificationSubmissionLogRepository{db: db}
}

// Create 创建一条日志记录
func (r *gormVerificationSubmissionLogRepository) Create(ctx context.Context, log *models.VerificationSubmissionLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// BatchCreate 批量创建日志记录
func (r *gormVerificationSubmissionLogRepository) BatchCreate(ctx context.Context, logs []*models.VerificationSubmissionLog) error {
	return r.db.WithContext(ctx).Create(logs).Error
}

// CountTotalPhones 统计系统中可用手机号码的总数量（排除已注销）
func (r *gormVerificationSubmissionLogRepository) CountTotalPhones(ctx context.Context) (int, error) {
	var count int64

	// 统计mobile_numbers表中除"已注销"状态外的所有手机号码数量
	err := r.db.WithContext(ctx).Model(&models.MobileNumber{}).
		Where("status != ?", "已注销").
		Count(&count).Error

	return int(count), err
}

// CountConfirmedPhones 统计已确认使用的手机号码数量
func (r *gormVerificationSubmissionLogRepository) CountConfirmedPhones(ctx context.Context) (int, error) {
	var count int64

	// 使用子查询获取每个手机号的最新操作
	subQuery := r.db.Model(&models.VerificationSubmissionLog{}).
		Select("phone_number, MAX(created_at) as latest_time").
		Group("phone_number")

	// 主查询统计最新操作为"confirm_usage"的手机号数量
	err := r.db.WithContext(ctx).Table("(?) as latest", subQuery).
		Joins("JOIN verification_submissions_log vsl ON vsl.phone_number = latest.phone_number AND vsl.created_at = latest.latest_time").
		Where("vsl.action_type = ?", models.ActionConfirmUsage).
		Count(&count).Error

	return int(count), err
}

// CountReportedIssuePhones 统计有问题的手机号码数量
func (r *gormVerificationSubmissionLogRepository) CountReportedIssuePhones(ctx context.Context) (int, error) {
	var count int64

	// 使用子查询获取每个手机号的最新操作
	subQuery := r.db.Model(&models.VerificationSubmissionLog{}).
		Select("phone_number, MAX(created_at) as latest_time").
		Group("phone_number")

	// 主查询统计最新操作为"report_issue"的手机号数量
	err := r.db.WithContext(ctx).Table("(?) as latest", subQuery).
		Joins("JOIN verification_submissions_log vsl ON vsl.phone_number = latest.phone_number AND vsl.created_at = latest.latest_time").
		Where("vsl.action_type = ?", models.ActionReportIssue).
		Count(&count).Error

	return int(count), err
}

// CountPendingPhones 统计待确认的手机号码数量
func (r *gormVerificationSubmissionLogRepository) CountPendingPhones(ctx context.Context) (int, error) {
	var count int64

	// 查询所有在系统中但未在日志表中有记录的手机号
	err := r.db.WithContext(ctx).Model(&models.MobileNumber{}).
		Where("phone_number NOT IN (SELECT DISTINCT phone_number FROM verification_submissions_log)").
		Count(&count).Error

	return int(count), err
}

// CountNewlyReportedPhones 统计新上报的手机号码数量
func (r *gormVerificationSubmissionLogRepository) CountNewlyReportedPhones(ctx context.Context) (int, error) {
	var count int64

	// 统计类型为"report_unlisted"的不同手机号数量
	err := r.db.WithContext(ctx).Model(&models.VerificationSubmissionLog{}).
		Where("action_type = ?", models.ActionReportUnlisted).
		Distinct("phone_number").
		Count(&count).Error

	return int(count), err
}

// FindLatestActionsByPhoneNumber 查询每个手机号的最新操作类型
func (r *gormVerificationSubmissionLogRepository) FindLatestActionsByPhoneNumber(ctx context.Context) (map[string]models.VerificationActionType, error) {
	// 使用原生SQL查询获取每个手机号的最新操作记录
	type Result struct {
		PhoneNumber string
		ActionType  models.VerificationActionType
	}

	var results []Result

	// 子查询获取每个手机号的最新记录
	subQuery := r.db.Model(&models.VerificationSubmissionLog{}).
		Select("phone_number, MAX(created_at) as latest_time").
		Group("phone_number")

	// 主查询获取对应的操作类型
	err := r.db.WithContext(ctx).Table("(?) as latest", subQuery).
		Select("vsl.phone_number, vsl.action_type").
		Joins("JOIN verification_submissions_log vsl ON vsl.phone_number = latest.phone_number AND vsl.created_at = latest.latest_time").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// 转换为map
	actionsMap := make(map[string]models.VerificationActionType, len(results))
	for _, r := range results {
		actionsMap[r.PhoneNumber] = r.ActionType
	}

	return actionsMap, nil
}

// FindConfirmedPhoneDetails 查询已确认使用的手机号码详情
func (r *gormVerificationSubmissionLogRepository) FindConfirmedPhoneDetails(ctx context.Context) ([]models.ConfirmedPhoneDetail, error) {
	// 子查询获取每个手机号的最新操作记录
	subQuery := r.db.Model(&models.VerificationSubmissionLog{}).
		Select("phone_number, MAX(created_at) as latest_time").
		Group("phone_number")

	// 主查询获取最新操作为"confirm_usage"的手机号详情
	var results []struct {
		ID          uint
		PhoneNumber string
		Department  *string
		CurrentUser string
		Purpose     *string
		ConfirmedBy string
		ConfirmedAt time.Time
	}

	// 构建联合查询，获取手机号详情
	err := r.db.WithContext(ctx).Table("(?) as latest", subQuery).
		Select("mn.id, mn.phone_number, e_current.department, e_current.full_name as current_user, mn.purpose, "+
			"e_confirmed.full_name as confirmed_by, vsl.created_at as confirmed_at").
		Joins("JOIN verification_submissions_log vsl ON vsl.phone_number = latest.phone_number AND vsl.created_at = latest.latest_time").
		Joins("JOIN mobile_numbers mn ON mn.phone_number = vsl.phone_number").
		Joins("JOIN employees e_confirmed ON e_confirmed.employee_id = vsl.employee_id").
		Joins("LEFT JOIN employees e_current ON e_current.employee_id = mn.current_employee_id").
		Where("vsl.action_type = ?", models.ActionConfirmUsage).
		Order("vsl.created_at desc").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// 转换为响应数据结构
	details := make([]models.ConfirmedPhoneDetail, 0, len(results))
	for _, r := range results {
		var department string
		if r.Department != nil {
			department = *r.Department
		}

		details = append(details, models.ConfirmedPhoneDetail{
			ID:          r.ID,
			PhoneNumber: r.PhoneNumber,
			Department:  department,
			CurrentUser: r.CurrentUser,
			Purpose:     r.Purpose,
			ConfirmedBy: r.ConfirmedBy,
			ConfirmedAt: r.ConfirmedAt,
		})
	}

	return details, nil
}
