package services

import (
	"context"
	"encoding/json" // 用于序列化 RequestedScopeValues
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/phone_management/configs"
	"github.com/phone_management/internal/models"
	"github.com/phone_management/internal/repositories"
	"github.com/phone_management/pkg/email"
	"gorm.io/gorm"
)

// 定义一些服务层特定的错误，如果需要的话
var ErrEmailDispatchFailed = errors.New("邮件发送失败")
var ErrBatchTaskNotFound = errors.New("批处理任务未找到")
var ErrTokenNotFound = errors.New("验证令牌不存在")
var ErrTokenExpired = errors.New("验证令牌已过期")

// 号码确认信息结构
type VerificationInfoResponse struct {
	EmployeeName    string           `json:"employeeName"`
	TokenValidUntil time.Time        `json:"tokenValidUntil"`
	NumbersToVerify []NumberToVerify `json:"numbersToVerify"`
}

// 待确认的号码信息
type NumberToVerify struct {
	MobileNumberId        uint   `json:"mobileNumberId"`
	PhoneNumber           string `json:"phoneNumber"`
	CurrentStatusInSystem string `json:"currentStatusInSystem"`
}

// VerificationService 定义了号码验证服务的接口
type VerificationService interface {
	// InitiateVerificationProcess 启动一个新的验证批处理任务，并返回批处理ID
	InitiateVerificationProcess(ctx context.Context, scopeType models.VerificationScopeType, scopeValues []string, durationDays int) (batchID string, err error)
	// GetVerificationBatchStatus 获取指定批处理任务的当前状态和统计信息
	GetVerificationBatchStatus(ctx context.Context, batchID string) (*models.VerificationBatchTask, error)
	// GetVerificationInfo 获取待确认的号码信息
	GetVerificationInfo(ctx context.Context, token string) (*models.VerificationInfo, error)
	// SubmitVerificationResult 提交号码确认结果
	SubmitVerificationResult(ctx context.Context, token string, request *models.VerificationSubmission) error
	// ProcessVerificationBatch (内部方法，可不由接口暴露，或仅为测试暴露)
	// processVerificationBatch(batchID string) // 改为非导出，由 InitiateVerificationProcess 内部 goroutine 调用
}

// verificationService 结构体现已包含 appConfig
type verificationService struct {
	employeeRepo          repositories.EmployeeRepository
	verificationTokenRepo repositories.VerificationTokenRepository
	batchTaskRepo         repositories.VerificationBatchTaskRepository // 新增批处理任务仓库
	mobileNumberRepo      repositories.MobileNumberRepository          // 新增手机号码仓库
	userReportedIssueRepo repositories.UserReportedIssueRepository     // 新增用户报告问题仓库
	appConfig             *configs.Configuration
}

// NewVerificationService 构造函数现已注入 appConfig
func NewVerificationService(employeeRepo repositories.EmployeeRepository, verificationTokenRepo repositories.VerificationTokenRepository, batchTaskRepo repositories.VerificationBatchTaskRepository, mobileNumberRepo repositories.MobileNumberRepository, userReportedIssueRepo repositories.UserReportedIssueRepository) VerificationService {
	return &verificationService{
		employeeRepo:          employeeRepo,
		verificationTokenRepo: verificationTokenRepo,
		batchTaskRepo:         batchTaskRepo,
		mobileNumberRepo:      mobileNumberRepo,
		userReportedIssueRepo: userReportedIssueRepo,
		appConfig:             &configs.AppConfig,
	}
}

// GetVerificationInfo 获取验证信息
func (s *verificationService) GetVerificationInfo(ctx context.Context, token string) (*models.VerificationInfo, error) {
	// 1. 验证token的有效性
	verificationToken, err := s.verificationTokenRepo.FindByToken(ctx, token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("查询验证令牌失败: %w", err)
	}

	// 2. 检查token状态
	if verificationToken.Status != models.VerificationTokenStatusPending {
		return nil, ErrTokenExpired
	}

	// 3. 检查token是否过期
	if time.Now().After(verificationToken.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// 4. 获取用户信息
	user, err := s.employeeRepo.GetEmployeeByEmployeeID(verificationToken.EmployeeID)
	if err != nil {
		return nil, fmt.Errorf("查询用户信息失败: %w", err)
	}

	// 5. 获取需要确认的号码列表
	mobileNumbers, err := s.mobileNumberRepo.FindAssignedToEmployee(ctx, verificationToken.EmployeeID)
	if err != nil {
		return nil, fmt.Errorf("获取号码列表失败: %w", err)
	}

	// 获取已报告问题的号码ID列表
	reportedIssueNumberIds, err := s.userReportedIssueRepo.FindReportedMobileNumberIdsByTokenId(ctx, verificationToken.ID)
	if err != nil {
		return nil, fmt.Errorf("获取报告问题号码列表失败: %w", err)
	}

	// 获取已确认使用的号码ID列表
	confirmedNumberIds, err := s.mobileNumberRepo.FindConfirmedNumberIdsByTokenId(ctx, verificationToken.ID)
	if err != nil {
		return nil, fmt.Errorf("获取已确认号码列表失败: %w", err)
	}

	// 获取用户通过此token已报告的未列出号码
	previouslyReported, err := s.userReportedIssueRepo.FindUnlistedByTokenId(ctx, verificationToken.ID)
	if err != nil {
		// 如果查询出错，可以记录日志，但为了不阻塞主流程，这里暂时不直接返回错误
		// 或者根据业务需求决定是否必须返回错误
		fmt.Printf("获取已报告的未列出号码失败: %v (token_id: %d)\n", err, verificationToken.ID)
		// previouslyReported 将保持为 nil 或空切片
	}

	// 转换为响应格式
	phoneNumbers := make([]models.VerificationPhoneNumber, 0, len(mobileNumbers))
	for _, number := range mobileNumbers {
		status := "pending"

		// 检查号码是否已被报告问题
		for _, reportedId := range reportedIssueNumberIds {
			if reportedId == number.ID {
				status = "reported"
				break
			}
		}

		// 检查号码是否已确认使用
		if status == "pending" {
			for _, confirmedId := range confirmedNumberIds {
				if confirmedId == number.ID {
					status = "confirmed"
					break
				}
			}
		}

		// 设置部门信息
		department := ""
		if user.Department != nil {
			department = *user.Department
		}

		phoneNumbers = append(phoneNumbers, models.VerificationPhoneNumber{
			ID:          number.ID,
			PhoneNumber: number.PhoneNumber,
			Department:  department,
			Status:      status,
		})
	}

	// 转换 previouslyReported 为响应格式
	reportedUnlistedInfos := make([]models.ReportedUnlistedNumberInfo, 0, len(previouslyReported))
	for _, reportedIssue := range previouslyReported {
		info := models.ReportedUnlistedNumberInfo{
			ReportedAt: reportedIssue.CreatedAt, // 使用报告问题记录的创建时间
		}
		if reportedIssue.ReportedPhoneNumber != nil {
			info.PhoneNumber = *reportedIssue.ReportedPhoneNumber
		}
		if reportedIssue.UserComment != nil {
			info.UserComment = *reportedIssue.UserComment
		}
		reportedUnlistedInfos = append(reportedUnlistedInfos, info)
	}

	return &models.VerificationInfo{
		EmployeeID:                 user.EmployeeID,
		EmployeeName:               user.FullName,
		PhoneNumbers:               phoneNumbers,
		PreviouslyReportedUnlisted: reportedUnlistedInfos, // 填充新字段
		ExpiresAt:                  verificationToken.ExpiresAt,
	}, nil
}

// GetVerificationBatchStatus 获取批处理任务的状态
func (s *verificationService) GetVerificationBatchStatus(ctx context.Context, batchID string) (*models.VerificationBatchTask, error) {
	task, err := s.batchTaskRepo.GetByID(ctx, batchID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // 假设 gorm 在这里
			return nil, ErrBatchTaskNotFound
		}
		return nil, fmt.Errorf("获取批处理任务失败: %w", err)
	}
	return task, nil
}

// processVerificationBatch 是实际执行批量处理的内部方法
// 它将在一个单独的 goroutine 中运行
func (s *verificationService) processVerificationBatch(initialTask *models.VerificationBatchTask) {
	ctx := context.Background() // 为后台任务创建一个新的上下文
	batchID := initialTask.ID

	var employees []models.Employee
	var err error

	// 将 scopeValues (如果存在) 从 JSON 字符串转换回 []string
	var actualScopeValues []string
	if initialTask.RequestedScopeValues != nil && *initialTask.RequestedScopeValues != "" {
		if jsonErr := json.Unmarshal([]byte(*initialTask.RequestedScopeValues), &actualScopeValues); jsonErr != nil {
			fmt.Printf("处理批处理 %s 失败：无法解析 scopeValues: %v\n", batchID, jsonErr)
			_ = s.batchTaskRepo.UpdateCountsAndStatus(ctx, batchID, 0, 0, 0, 0, models.BatchTaskStatusFailed, &models.EmailFailureDetail{Reason: "无法解析请求参数"})
			return
		}
	}

	switch initialTask.RequestedScopeType {
	case models.VerificationScopeAllUsers:
		employees, err = s.employeeRepo.FindAllActive(ctx)
	case models.VerificationScopeDepartment:
		employees, err = s.employeeRepo.FindActiveByDepartmentNames(ctx, actualScopeValues)
	case models.VerificationScopeEmployeeIDs:
		employees, err = s.employeeRepo.FindActiveByEmployeeIDs(ctx, actualScopeValues)
	default:
		err = fmt.Errorf("无效的范围类型: %s", initialTask.RequestedScopeType)
	}

	if err != nil {
		fmt.Printf("处理批处理 %s 失败：查找员工失败: %v\n", batchID, err)
		_ = s.batchTaskRepo.UpdateCountsAndStatus(ctx, batchID, 0, 0, 0, 0, models.BatchTaskStatusFailed, &models.EmailFailureDetail{Reason: "查找目标员工失败: " + err.Error()})
		return
	}

	if len(employees) == 0 {
		fmt.Printf("处理批处理 %s：没有找到符合条件的员工\n", batchID)
		_ = s.batchTaskRepo.UpdateCountsAndStatus(ctx, batchID, 0, 0, 0, 0, models.BatchTaskStatusCompleted, nil) // 没有员工也算完成
		return
	}

	// 更新任务的总处理员工数 (如果创建时未设置或不准确)
	if initialTask.TotalEmployeesToProcess != len(employees) {
		initialTask.TotalEmployeesToProcess = len(employees)
		// 这可以是一个单独的 Update 调用，或者在第一次 UpdateCountsAndStatus 时包含
		_ = s.batchTaskRepo.Update(ctx, initialTask) // 更新总数
	}

	frontendBaseURL := s.appConfig.FrontendBaseURL
	var localTokensGenerated, localEmailsAttempted, localEmailsSucceeded, localEmailsFailed int

	for _, emp := range employees {
		token := uuid.NewString()
		expiresAt := time.Now().AddDate(0, 0, initialTask.RequestedDurationDays)
		verificationToken := &models.VerificationToken{
			EmployeeID: emp.EmployeeID,
			Token:      token,
			Status:     models.VerificationTokenStatusPending,
			ExpiresAt:  expiresAt,
		}

		createErr := s.verificationTokenRepo.Create(ctx, verificationToken)
		if createErr != nil {
			fmt.Printf("批处理 %s：为员工 %s 创建令牌失败: %v\n", batchID, emp.EmployeeID, createErr)
			// 即使令牌创建失败，也尝试记录为一次尝试（或根据业务逻辑定义）
			// 这里我们不直接更新数据库，而是在循环结束后批量更新或根据 newStatus 决定
			// 但需要记录这个失败，可能影响邮件发送的尝试
			// 简单起见，如果令牌创建失败，我们就不尝试发邮件了
			_ = s.batchTaskRepo.UpdateCountsAndStatus(ctx, batchID, 0, 0, 0, 1, models.BatchTaskStatusInProgress, &models.EmailFailureDetail{
				EmployeeID: emp.EmployeeID, EmployeeName: emp.FullName, EmailAddress: "N/A (Token creation failed)", Reason: "Token creation failed: " + createErr.Error(),
			})
			localEmailsFailed++ // 算作邮件处理失败的一部分
			continue            // 继续处理下一个员工
		}
		localTokensGenerated++

		if emp.Email != nil && *emp.Email != "" {
			localEmailsAttempted++
			verificationLink := fmt.Sprintf("%s/verify-numbers?token=%s", frontendBaseURL, token)
			sendErr := email.SendVerificationEmail(*emp.Email, emp.FullName, verificationLink)
			if sendErr != nil {
				fmt.Printf("批处理 %s：发送确认邮件给 %s (%s) 失败: %v\n", batchID, emp.FullName, *emp.Email, sendErr)
				localEmailsFailed++
				_ = s.batchTaskRepo.UpdateCountsAndStatus(ctx, batchID, 1, 1, 0, 1, models.BatchTaskStatusInProgress, &models.EmailFailureDetail{
					EmployeeID: emp.EmployeeID, EmployeeName: emp.FullName, EmailAddress: *emp.Email, Reason: sendErr.Error(),
				})
			} else {
				localEmailsSucceeded++
				_ = s.batchTaskRepo.UpdateCountsAndStatus(ctx, batchID, 1, 1, 1, 0, models.BatchTaskStatusInProgress, nil)
			}
		} else {
			fmt.Printf("批处理 %s：员工 %s (%s) 缺少邮箱地址，跳过发送确认邮件\n", batchID, emp.FullName, emp.EmployeeID)
			// 这种情况也算作一次尝试，但失败了（因为没有邮箱）
			localEmailsAttempted++
			localEmailsFailed++
			_ = s.batchTaskRepo.UpdateCountsAndStatus(ctx, batchID, 1, 1, 0, 1, models.BatchTaskStatusInProgress, &models.EmailFailureDetail{
				EmployeeID: emp.EmployeeID, EmployeeName: emp.FullName, EmailAddress: "N/A", Reason: "Missing email address",
			})
		}
		// 这里的 UpdateCountsAndStatus 是在每次循环中调用，对于大量员工可能导致频繁DB操作。
		// 更优化的方式是本地累积计数，然后定期（例如每 N 个员工或每隔 X 秒）或在最后统一更新数据库。
		// 当前的简化实现是每次都更新。
	}

	// 所有员工处理完毕，决定最终状态
	finalStatus := models.BatchTaskStatusCompleted
	if localEmailsFailed > 0 {
		finalStatus = models.BatchTaskStatusCompletedWithErrors
	}
	// 最后一次更新，确保所有计数准确，并设置最终状态
	// 注意：如果之前的 UpdateCountsAndStatus 已经是原子增量，这里可能只需要更新最终状态和 ErrorSummary (如果 ErrorSummary 是累积的)
	// 当前 UpdateCountsAndStatus 已经是增量，所以这里主要是设置最终状态
	// 为了简化，我们假设最后一次调用 UpdateCountsAndStatus 来设置状态，并不再传递增量计数（因为它们已在循环中处理）
	// 更好的做法是服务层维护 task 对象，并在最后调用一次 taskRepo.Update(ctx, task)
	finalUpdateErr := s.batchTaskRepo.UpdateCountsAndStatus(ctx, batchID, 0, 0, 0, 0, finalStatus, nil) // 只更新状态
	if finalUpdateErr != nil {
		fmt.Printf("批处理 %s：更新最终状态失败: %v\n", batchID, finalUpdateErr)
	}
	fmt.Printf("批处理 %s 完成。总员工: %d, 令牌生成: %d, 邮件尝试: %d, 成功: %d, 失败: %d. 最终状态: %s\n",
		batchID, len(employees), localTokensGenerated, localEmailsAttempted, localEmailsSucceeded, localEmailsFailed, finalStatus)
}

// InitiateVerificationProcess 创建一个新的批处理任务并异步启动它
func (s *verificationService) InitiateVerificationProcess(ctx context.Context, scopeType models.VerificationScopeType, scopeValues []string, durationDays int) (batchID string, err error) {
	// 1. 查找员工 (预检查，获取总数，但不在这里处理每个员工的细节)
	// 这一步主要是为了得到 TotalEmployeesToProcess 的初始值和校验请求是否有效
	var preliminaryEmployees []models.Employee
	switch scopeType {
	case models.VerificationScopeAllUsers:
		preliminaryEmployees, err = s.employeeRepo.FindAllActive(ctx)
	case models.VerificationScopeDepartment:
		if len(scopeValues) == 0 {
			return "", fmt.Errorf("部门名称列表不能为空")
		}
		preliminaryEmployees, err = s.employeeRepo.FindActiveByDepartmentNames(ctx, scopeValues)
	case models.VerificationScopeEmployeeIDs:
		if len(scopeValues) == 0 {
			return "", fmt.Errorf("员工ID列表不能为空")
		}
		preliminaryEmployees, err = s.employeeRepo.FindActiveByEmployeeIDs(ctx, scopeValues)
	default:
		return "", fmt.Errorf("无效的范围类型: %s", scopeType)
	}

	if err != nil {
		return "", fmt.Errorf("查找目标员工失败: %w", err)
	}
	if len(preliminaryEmployees) == 0 {
		// 如果没有员工，可以不创建批处理任务，或者创建一个状态为Completed的空任务
		return "", fmt.Errorf("没有找到符合条件的员工来发起确认流程")
	}

	// 序列化 scopeValues 以便存储
	var scopeValuesJSON *string
	if len(scopeValues) > 0 {
		jsonBytes, jsonErr := json.Marshal(scopeValues)
		if jsonErr != nil {
			return "", fmt.Errorf("序列化 scopeValues 失败: %w", jsonErr)
		}
		s := string(jsonBytes)
		scopeValuesJSON = &s
	}

	// 2. 创建 VerificationBatchTask 记录
	newTask := &models.VerificationBatchTask{
		// ID 会在 BeforeCreate hook 中生成
		Status:                  models.BatchTaskStatusPending,
		TotalEmployeesToProcess: len(preliminaryEmployees),
		TokensGeneratedCount:    0,
		EmailsAttemptedCount:    0,
		EmailsSucceededCount:    0,
		EmailsFailedCount:       0,
		RequestedScopeType:      scopeType,
		RequestedScopeValues:    scopeValuesJSON,
		RequestedDurationDays:   durationDays,
	}

	if err := s.batchTaskRepo.Create(ctx, newTask); err != nil {
		return "", fmt.Errorf("创建批处理任务失败: %w", err)
	}

	// 3. 异步启动 processVerificationBatch
	go s.processVerificationBatch(newTask) // 传递新创建的任务对象，确保ID和其他初始值可用

	return newTask.ID, nil
}

// SubmitVerificationResult 提交号码确认结果
func (s *verificationService) SubmitVerificationResult(ctx context.Context, token string, request *models.VerificationSubmission) error {
	// 1. 验证token的有效性
	verificationToken, err := s.verificationTokenRepo.FindByToken(ctx, token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTokenNotFound
		}
		return fmt.Errorf("查询验证令牌失败: %w", err)
	}

	// 2. 检查token状态
	if verificationToken.Status != models.VerificationTokenStatusPending {
		return ErrTokenExpired
	}

	// 3. 检查token是否过期
	if time.Now().After(verificationToken.ExpiresAt) {
		return ErrTokenExpired
	}

	// 4. 处理verified numbers
	for _, verifiedNumber := range request.VerifiedNumbers {
		switch verifiedNumber.Action {
		case "confirm_usage":
			if err := s.mobileNumberRepo.UpdateLastConfirmationDate(ctx, verifiedNumber.MobileNumberId); err != nil {
				return fmt.Errorf("更新号码确认日期失败: %w", err)
			}
		case "report_issue":
			if err := s.mobileNumberRepo.MarkAsReportedByUser(ctx, verifiedNumber.MobileNumberId); err != nil {
				return fmt.Errorf("标记号码为用户报告问题失败: %w", err)
			}

			// 创建用户报告问题记录
			reportedIssue := &models.UserReportedIssue{
				VerificationTokenId:  &verificationToken.ID,
				ReportedByEmployeeID: verificationToken.EmployeeID,
				MobileNumberDbId:     &verifiedNumber.MobileNumberId,
				IssueType:            "number_issue",
				UserComment:          &verifiedNumber.UserComment,
				AdminActionStatus:    "pending_review",
			}

			if err := s.userReportedIssueRepo.CreateReportedIssue(ctx, reportedIssue); err != nil {
				return fmt.Errorf("创建用户报告问题记录失败: %w", err)
			}
		default:
			return fmt.Errorf("无效的操作类型: %s", verifiedNumber.Action)
		}
	}

	// 5. 处理unlisted numbers
	for _, unlistedNumber := range request.UnlistedNumbersReported {
		reportedIssue := &models.UserReportedIssue{
			VerificationTokenId:  &verificationToken.ID,
			ReportedByEmployeeID: verificationToken.EmployeeID,
			ReportedPhoneNumber:  &unlistedNumber.PhoneNumber,
			IssueType:            "unlisted_number",
			UserComment:          &unlistedNumber.UserComment,
			AdminActionStatus:    "pending_review",
		}

		if err := s.userReportedIssueRepo.CreateReportedIssue(ctx, reportedIssue); err != nil {
			return fmt.Errorf("创建未列出号码报告记录失败: %w", err)
		}
	}

	// 不再更新令牌状态为used，保持pending状态直到过期
	return nil
}
