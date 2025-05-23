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
	// GetPhoneVerificationStatus 获取基于手机号码维度的管理员视图
	GetPhoneVerificationStatus(ctx context.Context, employeeID, departmentName string) (*models.PhoneVerificationStatusResponse, error)
	// ProcessVerificationBatch (内部方法，可不由接口暴露，或仅为测试暴露)
	// processVerificationBatch(batchID string) // 改为非导出，由 InitiateVerificationProcess 内部 goroutine 调用
}

// verificationService 结构体现已包含 appConfig
type verificationService struct {
	employeeRepo          repositories.EmployeeRepository
	verificationTokenRepo repositories.VerificationTokenRepository
	batchTaskRepo         repositories.VerificationBatchTaskRepository     // 新增批处理任务仓库
	mobileNumberRepo      repositories.MobileNumberRepository              // 新增手机号码仓库
	userReportedIssueRepo repositories.UserReportedIssueRepository         // 新增用户报告问题仓库
	submissionLogRepo     repositories.VerificationSubmissionLogRepository // 新增验证提交日志仓库
	appConfig             *configs.Configuration
	db                    *gorm.DB
}

// NewVerificationService 构造函数现已注入 appConfig
func NewVerificationService(employeeRepo repositories.EmployeeRepository, verificationTokenRepo repositories.VerificationTokenRepository, batchTaskRepo repositories.VerificationBatchTaskRepository, mobileNumberRepo repositories.MobileNumberRepository, userReportedIssueRepo repositories.UserReportedIssueRepository, submissionLogRepo repositories.VerificationSubmissionLogRepository, db *gorm.DB) VerificationService {
	return &verificationService{
		employeeRepo:          employeeRepo,
		verificationTokenRepo: verificationTokenRepo,
		batchTaskRepo:         batchTaskRepo,
		mobileNumberRepo:      mobileNumberRepo,
		userReportedIssueRepo: userReportedIssueRepo,
		submissionLogRepo:     submissionLogRepo,
		appConfig:             &configs.AppConfig,
		db:                    db,
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
		var userComment *string

		// 检查号码是否已被报告问题
		for _, reportedId := range reportedIssueNumberIds {
			if reportedId == number.ID {
				status = "reported"
				// 获取该号码最新的报告评论
				latestIssue, reportErr := s.userReportedIssueRepo.FindLatestByMobileNumberIdAndTokenId(ctx, number.ID, verificationToken.ID)
				if reportErr != nil {
					// 如果获取评论失败，可以记录日志，但程序应继续
					fmt.Printf("获取号码 %d (token %d) 的报告评论失败: %v\n", number.ID, verificationToken.ID, reportErr)
				} else if latestIssue != nil {
					userComment = latestIssue.UserComment
				}
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
			Purpose:     number.Purpose,
			Status:      status,
			UserComment: userComment, // 填充用户评论
		})
	}

	// 转换 previouslyReported 为响应格式
	reportedUnlistedInfos := make([]models.ReportedUnlistedNumberInfo, 0, len(previouslyReported))
	for _, reportedIssue := range previouslyReported {
		info := models.ReportedUnlistedNumberInfo{
			ReportedAt: reportedIssue.CreatedAt, // 使用报告问题记录的创建时间
			Purpose:    reportedIssue.Purpose,   // 填充 Purpose 字段
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

	// 收集需要创建的日志记录
	var submissionLogs []*models.VerificationSubmissionLog

	// 4. 处理verified numbers
	for _, verifiedNumber := range request.VerifiedNumbers {
		// 通过直接查询数据库获取手机号码信息
		var phoneNumber string

		// 使用数据库直接查询
		if err := s.db.Model(&models.MobileNumber{}).
			Where("id = ?", verifiedNumber.MobileNumberId).
			Select("phone_number").
			Scan(&phoneNumber).Error; err != nil {
			return fmt.Errorf("获取手机号码信息失败: %w", err)
		}

		// 创建日志记录
		actionType := models.ActionConfirmUsage
		if verifiedNumber.Action == "report_issue" {
			actionType = models.ActionReportIssue
		}

		submissionLog := &models.VerificationSubmissionLog{
			EmployeeID:          verificationToken.EmployeeID,
			VerificationTokenID: verificationToken.ID,
			MobileNumberID:      &verifiedNumber.MobileNumberId,
			PhoneNumber:         phoneNumber,
			ActionType:          actionType,
			Purpose:             verifiedNumber.Purpose,
			UserComment:         &verifiedNumber.UserComment,
		}
		submissionLogs = append(submissionLogs, submissionLog)

		switch verifiedNumber.Action {
		case "confirm_usage":
			if err := s.mobileNumberRepo.UpdateLastConfirmationDate(ctx, verifiedNumber.MobileNumberId); err != nil {
				return fmt.Errorf("更新号码确认日期失败: %w", err)
			}

			// 如果用户提供了新的用途信息，更新号码用途
			if verifiedNumber.Purpose != nil {
				updates := map[string]interface{}{
					"purpose": verifiedNumber.Purpose,
				}
				if _, err := s.mobileNumberRepo.UpdateMobileNumber(verifiedNumber.MobileNumberId, updates); err != nil {
					return fmt.Errorf("更新号码用途失败: %w", err)
				}
			}
		case "report_issue":
			if err := s.mobileNumberRepo.MarkAsReportedByUser(ctx, verifiedNumber.MobileNumberId); err != nil {
				return fmt.Errorf("标记号码为用户报告问题失败: %w", err)
			}

			// 检查是否存在由当前用户提交的、针对此号码的待处理报告
			existingIssue, findErr := s.userReportedIssueRepo.FindPendingByMobileNumberDbIdAndEmployeeId(ctx, verifiedNumber.MobileNumberId, verificationToken.EmployeeID)
			if findErr != nil {
				return fmt.Errorf("查找现有报告失败: %w", findErr)
			}

			var issueToSave *models.UserReportedIssue
			if existingIssue != nil {
				// 更新现有报告
				existingIssue.UserComment = &verifiedNumber.UserComment
				existingIssue.VerificationTokenId = &verificationToken.ID // 更新关联的token，以便追溯
				// Purpose 字段在此处不设置，因为它主要用于未列出号码的报告
				issueToSave = existingIssue
			} else {
				// 创建新报告
				issueToSave = &models.UserReportedIssue{
					VerificationTokenId:  &verificationToken.ID,
					ReportedByEmployeeID: verificationToken.EmployeeID,
					MobileNumberDbId:     &verifiedNumber.MobileNumberId,
					IssueType:            "number_issue",
					UserComment:          &verifiedNumber.UserComment,
					// Purpose 字段在此处不设置
					AdminActionStatus: "pending_review",
				}
			}

			if err := s.userReportedIssueRepo.SaveReportedIssue(ctx, issueToSave); err != nil {
				return fmt.Errorf("保存用户报告问题记录失败: %w", err)
			}
		default:
			return fmt.Errorf("无效的操作类型: %s", verifiedNumber.Action)
		}
	}

	// 5. 处理unlisted numbers
	for _, unlistedNumber := range request.UnlistedNumbersReported {
		// 创建日志记录
		submissionLog := &models.VerificationSubmissionLog{
			EmployeeID:          verificationToken.EmployeeID,
			VerificationTokenID: verificationToken.ID,
			MobileNumberID:      nil, // 未列出的号码没有系统ID
			PhoneNumber:         unlistedNumber.PhoneNumber,
			ActionType:          models.ActionReportUnlisted,
			Purpose:             unlistedNumber.Purpose,
			UserComment:         &unlistedNumber.UserComment,
		}
		submissionLogs = append(submissionLogs, submissionLog)

		// 检查是否存在由当前用户提交的、针对此未列出号码的待处理报告
		existingUnlistedIssue, findUnlistedErr := s.userReportedIssueRepo.FindPendingByReportedPhoneNumberAndEmployeeId(ctx, unlistedNumber.PhoneNumber, verificationToken.EmployeeID)
		if findUnlistedErr != nil {
			return fmt.Errorf("查找现有未列出号码报告失败: %w", findUnlistedErr)
		}

		var unlistedIssueToSave *models.UserReportedIssue
		if existingUnlistedIssue != nil {
			// 更新现有未列出号码报告
			existingUnlistedIssue.UserComment = &unlistedNumber.UserComment
			existingUnlistedIssue.Purpose = unlistedNumber.Purpose // 更新 Purpose
			existingUnlistedIssue.VerificationTokenId = &verificationToken.ID
			unlistedIssueToSave = existingUnlistedIssue
		} else {
			// 创建新的未列出号码报告
			unlistedIssueToSave = &models.UserReportedIssue{
				VerificationTokenId:  &verificationToken.ID,
				ReportedByEmployeeID: verificationToken.EmployeeID,
				ReportedPhoneNumber:  &unlistedNumber.PhoneNumber,
				IssueType:            "unlisted_number",
				UserComment:          &unlistedNumber.UserComment,
				Purpose:              unlistedNumber.Purpose, // 保存 Purpose
				AdminActionStatus:    "pending_review",
			}
		}

		if err := s.userReportedIssueRepo.SaveReportedIssue(ctx, unlistedIssueToSave); err != nil {
			return fmt.Errorf("保存未列出号码报告记录失败: %w", err)
		}
	}

	// 批量创建提交日志
	if len(submissionLogs) > 0 {
		if err := s.submissionLogRepo.BatchCreate(ctx, submissionLogs); err != nil {
			return fmt.Errorf("创建提交日志记录失败: %w", err)
		}
	}

	// 不再更新令牌状态为used，保持pending状态直到过期
	return nil
}

// GetPhoneVerificationStatus 获取基于手机号码维度的管理员视图
func (s *verificationService) GetPhoneVerificationStatus(ctx context.Context, employeeID, departmentName string) (*models.PhoneVerificationStatusResponse, error) {
	response := &models.PhoneVerificationStatusResponse{}

	// 1. 获取统计摘要
	// 1.0 计算系统中可用手机号码总数量（排除已注销）
	totalCount, err := s.submissionLogRepo.CountTotalPhones(ctx)
	if err != nil {
		return nil, fmt.Errorf("计算手机号码总数量失败: %w", err)
	}

	// 1.1 计算已确认使用的手机号码数
	confirmedCount, err := s.submissionLogRepo.CountConfirmedPhones(ctx)
	if err != nil {
		return nil, fmt.Errorf("计算已确认使用的手机号码数失败: %w", err)
	}

	// 1.2 计算有问题的手机号码数
	reportedIssuesCount, err := s.submissionLogRepo.CountReportedIssuePhones(ctx)
	if err != nil {
		return nil, fmt.Errorf("计算有问题的手机号码数失败: %w", err)
	}

	// 1.3 计算待确认的手机号码数
	pendingCount, err := s.submissionLogRepo.CountPendingPhones(ctx)
	if err != nil {
		return nil, fmt.Errorf("计算待确认的手机号码数失败: %w", err)
	}

	// 1.4 计算新上报的手机号码数
	newlyReportedCount, err := s.submissionLogRepo.CountNewlyReportedPhones(ctx)
	if err != nil {
		return nil, fmt.Errorf("计算新上报的手机号码数失败: %w", err)
	}

	// 1.5 获取已确认使用的手机号码详情列表
	confirmedPhones, err := s.submissionLogRepo.FindConfirmedPhoneDetails(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取已确认使用的手机号码详情列表失败: %w", err)
	}
	response.ConfirmedPhones = confirmedPhones

	// 构建统计摘要
	response.Summary = models.PhoneVerificationSummary{
		TotalPhonesCount:         totalCount,
		ConfirmedPhonesCount:     confirmedCount,
		ReportedIssuesCount:      reportedIssuesCount,
		PendingPhonesCount:       pendingCount,
		NewlyReportedPhonesCount: newlyReportedCount,
	}

	// 2. 获取未响应用户列表
	pendingUsers, err := s.verificationTokenRepo.FindPendingTokensWithEmployeeInfo(ctx, employeeID, departmentName)
	if err != nil {
		return nil, fmt.Errorf("获取未响应用户列表失败: %w", err)
	}
	response.PendingUsers = pendingUsers

	// 3. 获取用户报告的问题列表
	reportedIssues, err := s.userReportedIssueRepo.FindReportedIssuesWithDetails(ctx, employeeID, departmentName)
	if err != nil {
		return nil, fmt.Errorf("获取用户报告的问题列表失败: %w", err)
	}
	response.ReportedIssues = reportedIssues

	// 4. 获取用户报告的未列出号码列表
	unlistedNumbers, err := s.userReportedIssueRepo.FindUnlistedNumbersWithDetails(ctx, employeeID, departmentName)
	if err != nil {
		return nil, fmt.Errorf("获取用户报告的未列出号码列表失败: %w", err)
	}
	response.UnlistedNumbers = unlistedNumbers

	return response, nil
}
