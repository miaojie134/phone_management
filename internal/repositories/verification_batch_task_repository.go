package repositories

import (
	"context"
	"encoding/json"
	"time"

	"github.com/phone_management/internal/models"
	"gorm.io/gorm"
)

// VerificationBatchTaskRepository 定义了批处理任务仓库的接口
type VerificationBatchTaskRepository interface {
	Create(ctx context.Context, task *models.VerificationBatchTask) error
	GetByID(ctx context.Context, batchID string) (*models.VerificationBatchTask, error)
	Update(ctx context.Context, task *models.VerificationBatchTask) error
	// UpdateCountsAndStatus 允许原子性地更新计数器和状态，或按需部分更新
	UpdateCountsAndStatus(ctx context.Context, batchID string,
		tokensToAdd int, emailsAttemptedToAdd int, emailsSucceededToAdd int, emailsFailedToAdd int,
		newStatus models.VerificationBatchTaskStatus, errorDetail *models.EmailFailureDetail) error
}

type gormVerificationBatchTaskRepository struct {
	db *gorm.DB
}

// NewGormVerificationBatchTaskRepository 创建一个新的 GORM 批处理任务仓库实例
func NewGormVerificationBatchTaskRepository(db *gorm.DB) VerificationBatchTaskRepository {
	return &gormVerificationBatchTaskRepository{db: db}
}

// Create 在数据库中创建一个新的批处理任务记录
func (r *gormVerificationBatchTaskRepository) Create(ctx context.Context, task *models.VerificationBatchTask) error {
	return r.db.WithContext(ctx).Create(task).Error
}

// GetByID 从数据库中按 ID 获取批处理任务
func (r *gormVerificationBatchTaskRepository) GetByID(ctx context.Context, batchID string) (*models.VerificationBatchTask, error) {
	var task models.VerificationBatchTask
	if err := r.db.WithContext(ctx).Where("id = ?", batchID).First(&task).Error; err != nil {
		return nil, err // 调用方应处理 gorm.ErrRecordNotFound
	}
	return &task, nil
}

// Update 更新数据库中的批处理任务记录 (通常用于最终状态更新)
func (r *gormVerificationBatchTaskRepository) Update(ctx context.Context, task *models.VerificationBatchTask) error {
	return r.db.WithContext(ctx).Save(task).Error
}

// UpdateCountsAndStatus 更新任务的计数和状态。
// 注意：这里的错误摘要更新逻辑做了简化，实际应用中可能需要更复杂的 JSON 数组追加逻辑。
// 对于高并发或精确计数，可能需要使用 gorm.Expr("tokens_generated_count + ?", tokensToAdd) 等原子操作。
func (r *gormVerificationBatchTaskRepository) UpdateCountsAndStatus(
	ctx context.Context, batchID string,
	tokensToAdd int, emailsAttemptedToAdd int, emailsSucceededToAdd int, emailsFailedToAdd int,
	newStatus models.VerificationBatchTaskStatus, errorDetail *models.EmailFailureDetail) error {

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task models.VerificationBatchTask
		if err := tx.Where("id = ?", batchID).First(&task).Error; err != nil {
			return err // 任务未找到
		}

		updates := map[string]interface{}{
			"tokens_generated_count": gorm.Expr("tokens_generated_count + ?", tokensToAdd),
			"emails_attempted_count": gorm.Expr("emails_attempted_count + ?", emailsAttemptedToAdd),
			"emails_succeeded_count": gorm.Expr("emails_succeeded_count + ?", emailsSucceededToAdd),
			"emails_failed_count":    gorm.Expr("emails_failed_count + ?", emailsFailedToAdd),
			"status":                 newStatus, // 直接设置新状态，或者在服务层决定最终状态
			"updated_at":             time.Now(),
		}

		// 简化错误摘要处理：简单地覆盖。在实际应用中，这可能需要解析现有JSON，追加，然后序列化回去。
		// 或者使用数据库特定的 JSON 操作函数。
		if errorDetail != nil {
			// 这是一个非常简化的示例，实际中你可能想把多个错误追加到一个JSON数组中
			// 这里我们假设 ErrorSummary 存储单个最新的错误详情（需要调整）或是一个简单的字符串拼接
			// 为了演示，我们先简单地将错误原因作为字符串存入，实际应采用更结构化的方式
			// 例如：json.Marshal([]models.EmailFailureDetail{*errorDetail}) 然后存储其字符串
			newErrorSummary := task.ErrorSummary
			jsonError, _ := json.Marshal(errorDetail)
			if newErrorSummary == nil || *newErrorSummary == "" {
				strJsonError := string(jsonError)
				newErrorSummary = &strJsonError
			} else {
				// 简单的追加示例，实际可能需要解析和合并JSON数组
				updatedSummary := *newErrorSummary + "\n" + string(jsonError)
				newErrorSummary = &updatedSummary
			}
			updates["error_summary"] = newErrorSummary
		}

		return tx.Model(&models.VerificationBatchTask{}).Where("id = ?", batchID).Updates(updates).Error
	})
}
