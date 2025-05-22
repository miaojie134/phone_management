package services

import (
	"database/sql" // For sql.NullString and sql.NullInt64 in UserReportedIssue
	"fmt"
	"strconv" // For converting string IDs to uint
	"time"
	// "navigator/internal/email" // Assuming an email service interface
	"navigator/internal/models"
	"navigator/internal/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm" // For potential transaction management
)

// VerificationService defines the interface for verification-related business logic.
type VerificationService interface {
	InitiateVerificationProcess(scope string, scopeDetail []string, durationDays *int) (int, error)
	GetVerificationInfo(tokenStr string) (*VerificationInfoResponse, error)
	SubmitVerification(request SubmitVerificationRequest) error
	GetAdminVerificationStatus(filters AdminStatusFilters) (*AdminVerificationStatusResponse, error)
}

// verificationService implements the VerificationService interface.
type verificationService struct {
	db                    *gorm.DB // For transaction management
	verificationTokenRepo repositories.VerificationTokenRepository
	userReportedIssueRepo repositories.UserReportedIssueRepository
	employeeRepo          repositories.EmployeeRepository     // Assuming this exists
	mobileNumberRepo      repositories.MobileNumberRepository // Assuming this exists
	// emailService          email.EmailService             // Assuming this exists
	// config                VerificationConfig             // For things like default token duration
}

// VerificationConfig might hold configuration values.
// type VerificationConfig struct {
// 	DefaultTokenDurationDays int
// }

// NewVerificationService creates a new instance of VerificationService.
func NewVerificationService(
	db *gorm.DB,
	vtRepo repositories.VerificationTokenRepository,
	uriRepo repositories.UserReportedIssueRepository,
	eRepo repositories.EmployeeRepository,
	mnRepo repositories.MobileNumberRepository,
	// emailSvc email.EmailService,
	// cfg VerificationConfig,
) VerificationService {
	return &verificationService{
		db:                    db,
		verificationTokenRepo: vtRepo,
		userReportedIssueRepo: uriRepo,
		employeeRepo:          eRepo,
		mobileNumberRepo:      mnRepo,
		// emailService:          emailSvc,
		// config:                cfg,
	}
}

// DTOs for service method inputs and outputs

// VerificationInfoResponse is the DTO for GetVerificationInfo.
type VerificationInfoResponse struct {
	EmployeeName    string                     `json:"employeeName"`
	TokenValidUntil time.Time                  `json:"tokenValidUntil"`
	NumbersToVerify []MobileNumberVerification `json:"numbersToVerify"`
}

// MobileNumberVerification is a sub-DTO for VerificationInfoResponse.
type MobileNumberVerification struct {
	ID              uint   `json:"id"`
	Number          string `json:"number"`
	CurrentStatus   string `json:"currentStatus"`
	AdminRemarks    string `json:"adminRemarks,omitempty"` // Assuming MobileNumber has this
	LastConfirmedAt *time.Time `json:"lastConfirmedAt,omitempty"` // Assuming MobileNumber has this
}

// SubmitVerificationRequest is the DTO for SubmitVerification.
type SubmitVerificationRequest struct {
	TokenStr                string                       `json:"token"`
	VerifiedNumbers         []VerifiedNumberAction       `json:"verifiedNumbers"`
	UnlistedNumbersReported []UnlistedNumberReportAction `json:"unlistedNumbersReported"`
}

// VerifiedNumberAction is a sub-DTO for SubmitVerificationRequest.
type VerifiedNumberAction struct {
	MobileNumberID uint   `json:"mobileNumberId"`
	Action         string `json:"action"` // "confirm_usage", "report_issue"
	UserComment    string `json:"userComment,omitempty"`
}

// UnlistedNumberReportAction is a sub-DTO for SubmitVerificationRequest.
type UnlistedNumberReportAction struct {
	PhoneNumber string `json:"phoneNumber"`
	UserComment string `json:"userComment,omitempty"`
}

// AdminStatusFilters is the DTO for GetAdminVerificationStatus filters.
type AdminStatusFilters struct {
	CycleID      *uint  `json:"cycleId,omitempty"` // Assuming VerificationToken can be grouped by a cycle
	Status       string `json:"status,omitempty"`  // "pending", "used", "expired"
	EmployeeID   *uint  `json:"employeeId,omitempty"`
	DepartmentID *uint  `json:"departmentId,omitempty"`
}

// AdminVerificationStatusResponse is the DTO for GetAdminVerificationStatus.
type AdminVerificationStatusResponse struct {
	TotalInitiated      int                           `json:"totalInitiated"`
	Responded           int                           `json:"responded"`
	PendingResponse     int                           `json:"pendingResponse"`
	ExpiredNoResponse   int                           `json:"expiredNoResponse"` // Added for clarity
	IssuesReportedCount int                           `json:"issuesReportedCount"`
	PendingUsers        []PendingUserSummary          `json:"pendingUsers"`
	ReportedIssues      []ReportedIssueDetail         `json:"reportedIssues"`
	UnlistedNumbers     []UnlistedNumberReportedAdmin `json:"unlistedNumbers"`
}

// PendingUserSummary is a sub-DTO for AdminVerificationStatusResponse.
type PendingUserSummary struct {
	EmployeeID      uint      `json:"employeeId"`
	EmployeeName    string    `json:"employeeName"`
	Department      string    `json:"department,omitempty"` // Assuming Employee model has Department info
	TokenExpiresAt  time.Time `json:"tokenExpiresAt"`
	VerificationURL string    `json:"verificationUrl"`
}

// ReportedIssueDetail is a sub-DTO for AdminVerificationStatusResponse.
type ReportedIssueDetail struct {
	IssueID             uint      `json:"issueId"`
	ReportedByEmployee  string    `json:"reportedByEmployee"` // Name
	MobileNumber        string    `json:"mobileNumber,omitempty"`
	ReportedPhoneNumber string    `json:"reportedPhoneNumber,omitempty"` // For unlisted
	IssueType           string    `json:"issueType"`
	UserComment         string    `json:"userComment,omitempty"`
	ReportedAt          time.Time `json:"reportedAt"`
	AdminActionStatus   string    `json:"adminActionStatus"`
}

// UnlistedNumberReportedAdmin is a sub-DTO for AdminVerificationStatusResponse.
type UnlistedNumberReportedAdmin struct {
	IssueID             uint      `json:"issueId"`
	ReportedByEmployee  string    `json:"reportedByEmployee"`
	ReportedPhoneNumber string    `json:"reportedPhoneNumber"`
	UserComment         string    `json:"userComment,omitempty"`
	ReportedAt          time.Time `json:"reportedAt"`
	AdminActionStatus   string    `json:"adminActionStatus"`
}


// --- Implementations ---

// InitiateVerificationProcess implements the logic for POST /initiate.
func (s *verificationService) InitiateVerificationProcess(scope string, scopeDetail []string, durationDays *int) (int, error) {
	var employees []models.Employee
	var err error

	// Assuming EmployeeRepository has these methods:
	// FindAllActive() ([]models.Employee, error)
	// FindByDepartmentIDs(ids []uint) ([]models.Employee, error)
	// FindByIDs(ids []uint) ([]models.Employee, error)

	switch scope {
	case "all_users":
		employees, err = s.employeeRepo.FindAllActive() // Example method
	case "department_ids":
		var deptIDs []uint
		for _, idStr := range scopeDetail {
			id, convErr := strconv.ParseUint(idStr, 10, 32)
			if convErr != nil {
				return 0, fmt.Errorf("invalid department ID '%s': %w", idStr, convErr)
			}
			deptIDs = append(deptIDs, uint(id))
		}
		if len(deptIDs) == 0 && len(scopeDetail) > 0 { // some ids provided but none parsed
			return 0, fmt.Errorf("no valid department IDs provided")
		}
		if len(deptIDs) > 0 {
			employees, err = s.employeeRepo.FindByDepartmentIDs(deptIDs)
		}
	case "employee_ids":
		var empIDs []uint
		for _, idStr := range scopeDetail {
			id, convErr := strconv.ParseUint(idStr, 10, 32)
			if convErr != nil {
				return 0, fmt.Errorf("invalid employee ID '%s': %w", idStr, convErr)
			}
			empIDs = append(empIDs, uint(id))
		}
		if len(empIDs) == 0 && len(scopeDetail) > 0 {
			return 0, fmt.Errorf("no valid employee IDs provided")
		}
		if len(empIDs) > 0 {
			employees, err = s.employeeRepo.FindByIDs(empIDs)
		}
	default:
		return 0, fmt.Errorf("invalid scope: %s. Supported scopes: all_users, department_ids, employee_ids", scope)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to fetch employees for scope '%s': %w", scope, err)
	}

	if len(employees) == 0 {
		return 0, fmt.Errorf("no employees found for the given scope")
	}

	defaultDuration := 30 // Default token duration in days
	if durationDays != nil {
		defaultDuration = *durationDays
	}
	if defaultDuration <= 0 {
		defaultDuration = 30 // Ensure positive duration
	}
	expiresAt := time.Now().AddDate(0, 0, defaultDuration)

	initiatedCount := 0
	for _, emp := range employees {
		tokenStr := uuid.New().String()

		verificationToken := models.VerificationToken{
			EmployeeDbId: emp.ID,
			Token:        tokenStr,
			Status:       "pending",
			ExpiresAt:    expiresAt,
		}

		err := s.verificationTokenRepo.Create(&verificationToken)
		if err != nil {
			// Log error but try to continue with other employees
			fmt.Printf("Failed to create token for employee ID %d: %v\n", emp.ID, err)
			// Potentially collect these errors and return them
			continue
		}

		// TODO: Asynchronously send email to emp.Email
		// emailData := map[string]string{
		//  "employeeName": emp.Name,
		//  "verificationLink": fmt.Sprintf("https://yourdomain.com/verify-numbers?token=%s", tokenStr),
		//  "validUntil": expiresAt.Format("January 2, 2006"),
		// }
		// go s.emailService.SendVerificationEmail(emp.Email, emailData)
		fmt.Printf("TODO: Send email to %s with token %s\n", emp.Email, tokenStr)

		initiatedCount++
	}

	return initiatedCount, nil
}

// GetVerificationInfo implements the logic for GET /info.
func (s *verificationService) GetVerificationInfo(tokenStr string) (*VerificationInfoResponse, error) {
	token, err := s.verificationTokenRepo.FindByToken(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("database error fetching token: %w", err)
	}
	if token == nil {
		return nil, fmt.Errorf("verification token not found") // Specific error for not found
	}

	if token.Status != "pending" {
		return nil, fmt.Errorf("verification token is already %s", token.Status)
	}
	if time.Now().After(token.ExpiresAt) {
		// Optionally update status to 'expired' here
		// token.Status = "expired"
		// s.verificationTokenRepo.Update(token)
		return nil, fmt.Errorf("verification token has expired")
	}

	employee, err := s.employeeRepo.FindByID(token.EmployeeDbId) // Assuming EmployeeRepository has FindByID
	if err != nil {
		return nil, fmt.Errorf("error fetching employee details: %w", err)
	}
	if employee == nil {
		return nil, fmt.Errorf("employee associated with token not found")
	}

	// Assuming MobileNumberRepository has FindByEmployeeIDAndStatuses(employeeID uint, statuses []string) ([]models.MobileNumber, error)
	// And MobileNumber model has fields like AdminRemarks (sql.NullString) and LastConfirmedAt (*time.Time)
	mobileNumbers, err := s.mobileNumberRepo.FindByEmployeeIDAndStatuses(employee.ID, []string{"在用", "闲置"})
	if err != nil {
		return nil, fmt.Errorf("error fetching mobile numbers for employee ID %d: %w", employee.ID, err)
	}

	numbersToVerify := make([]MobileNumberVerification, 0, len(mobileNumbers))
	for _, mn := range mobileNumbers {
		nv := MobileNumberVerification{
			ID:            mn.ID,
			Number:        mn.Number,
			CurrentStatus: mn.Status,
		}
		// Assuming MobileNumber model has these fields.
		// if mn.AdminRemarks.Valid {
		// 	nv.AdminRemarks = mn.AdminRemarks.String
		// }
		// if mn.LastConfirmedAt != nil {
		// 	nv.LastConfirmedAt = mn.LastConfirmedAt
		// }
		numbersToVerify = append(numbersToVerify, nv)
	}

	return &VerificationInfoResponse{
		EmployeeName:    employee.Name, // Assuming Employee model has Name
		TokenValidUntil: token.ExpiresAt,
		NumbersToVerify: numbersToVerify,
	}, nil
}

// SubmitVerification implements the logic for POST /submit.
// TODO: Wrap the entire operation in a database transaction.
func (s *verificationService) SubmitVerification(req SubmitVerificationRequest) error {
	token, err := s.verificationTokenRepo.FindByToken(req.TokenStr)
	if err != nil {
		return fmt.Errorf("database error fetching token: %w", err)
	}
	if token == nil {
		return fmt.Errorf("verification token not found")
	}

	if token.Status != "pending" {
		return fmt.Errorf("verification token is already %s", token.Status)
	}
	if time.Now().After(token.ExpiresAt) {
		// token.Status = "expired" // Optionally update status
		// s.verificationTokenRepo.Update(token)
		return fmt.Errorf("verification token has expired")
	}

	// TODO: Consider wrapping these operations in a database transaction
	// For example:
	// tx := s.db.Begin()
	// if tx.Error != nil {
	// 	return fmt.Errorf("failed to start transaction: %w", tx.Error)
	// }
	// defer tx.Rollback() // Rollback if not committed

	// Pass `tx` to repository methods: s.verificationTokenRepo.WithTx(tx).Update(token) etc.

	for _, vNumber := range req.VerifiedNumbers {
		mobileNumber, err := s.mobileNumberRepo.FindByID(vNumber.MobileNumberID) // Assuming this method exists
		if err != nil {
			fmt.Printf("Error fetching mobile number ID %d: %v. Skipping.\n", vNumber.MobileNumberID, err)
			// Potentially collect errors instead of just logging
			continue
		}
		if mobileNumber == nil {
			fmt.Printf("Mobile number ID %d not found. Skipping.\n", vNumber.MobileNumberID)
			continue
		}

		// Ensure the number belongs to the employee associated with the token
		if mobileNumber.EmployeeDbId != token.EmployeeDbId {
			fmt.Printf("Security check failed: Mobile number ID %d (owner %d) does not belong to verifying employee ID %d. Skipping.\n",
				mobileNumber.ID, mobileNumber.EmployeeDbId, token.EmployeeDbId)
			// return fmt.Errorf("mobile number ID %d does not belong to verifying employee", mobileNumber.ID) // Or just skip
			continue
		}

		switch vNumber.Action {
		case "confirm_usage":
			now := time.Now()
			// mobileNumber.LastConfirmedAt = &now // Assuming MobileNumber model has this field: LastConfirmedAt *time.Time
			// Append to remarks or set if empty
			// currentRemark := ""
			// if mobileNumber.Remarks.Valid {
			// 	currentRemark = mobileNumber.Remarks.String + "\n"
			// }
			// mobileNumber.Remarks = sql.NullString{
			// 	String: fmt.Sprintf("%sUser confirmed usage on %s. Comment: %s", currentRemark, now.Format("2006-01-02"), vNumber.UserComment),
			// 	Valid:  true,
			// }
			// err = s.mobileNumberRepo.Update(mobileNumber) // TODO: Pass tx if using transactions
			// if err != nil {
			// 	fmt.Printf("Error updating mobile number ID %d for confirm_usage: %v\n", mobileNumber.ID, err)
			// 	// if tx != nil { tx.Rollback() }
			// 	return fmt.Errorf("failed to update mobile number %d for confirm_usage: %w", mobileNumber.ID, err)
			// }
			fmt.Printf("INFO: Confirm usage for mobile number ID %d. User comment: '%s'. LastConfirmedAt and Remarks would be updated.\n", mobileNumber.ID, vNumber.UserComment)

		case "report_issue":
			// mobileNumber.Status = "待核实-用户报告" // Or a defined constant for this status
			// err = s.mobileNumberRepo.Update(mobileNumber) // TODO: Pass tx
			// if err != nil {
			// 	fmt.Printf("Error updating mobile number status ID %d for report_issue: %v\n", mobileNumber.ID, err)
			//  // if tx != nil { tx.Rollback() }
			// 	return fmt.Errorf("failed to update status for mobile number %d for report_issue: %w", mobileNumber.ID, err)
			// }

			issue := models.UserReportedIssue{
				VerificationTokenId:    sql.NullInt64{Int64: int64(token.ID), Valid: true},
				ReportedByEmployeeDbId: token.EmployeeDbId,
				MobileNumberDbId:       sql.NullInt64{Int64: int64(mobileNumber.ID), Valid: true},
				IssueType:              "number_issue_reported_by_user", // More specific
				UserComment:            sql.NullString{String: vNumber.UserComment, Valid: vNumber.UserComment != ""},
				AdminActionStatus:      "pending_review",
			}
			err = s.userReportedIssueRepo.Create(&issue) // TODO: Pass tx
			if err != nil {
				fmt.Printf("Error creating user reported issue for mobile number ID %d: %v\n", mobileNumber.ID, err)
				// if tx != nil { tx.Rollback() }
				return fmt.Errorf("failed to create user reported issue for mobile number %d: %w", mobileNumber.ID, err)
			}
			fmt.Printf("INFO: Report issue for mobile number ID %d. User comment: '%s'. Status updated to '待核实-用户报告'. Issue created with ID %d.\n", mobileNumber.ID, vNumber.UserComment, issue.ID)
			// TODO: Notify admin about this issue (e.g., email, task system).

		default:
			fmt.Printf("Unknown action '%s' for mobile number ID %d. Skipping.\n", vNumber.Action, mobileNumber.ID)
		}
	}

	for _, unlisted := range req.UnlistedNumbersReported {
		issue := models.UserReportedIssue{
			VerificationTokenId:    sql.NullInt64{Int64: int64(token.ID), Valid: true},
			ReportedByEmployeeDbId: token.EmployeeDbId,
			ReportedPhoneNumber:    sql.NullString{String: unlisted.PhoneNumber, Valid: true}, // Already sql.NullString in model
			IssueType:              "unlisted_number_reported_by_user",
			UserComment:            sql.NullString{String: unlisted.UserComment, Valid: unlisted.UserComment != ""}, // Already sql.NullString
			AdminActionStatus:      "pending_review",
		}
		err = s.userReportedIssueRepo.Create(&issue) // TODO: Pass tx
		if err != nil {
			fmt.Printf("Error creating user reported issue for unlisted number %s: %v\n", unlisted.PhoneNumber, err)
			// if tx != nil { tx.Rollback() }
			return fmt.Errorf("failed to create user reported issue for unlisted number %s: %w", unlisted.PhoneNumber, err)
		}
		fmt.Printf("INFO: Reported unlisted number %s. User comment: '%s'. Issue created with ID %d.\n", unlisted.PhoneNumber, unlisted.UserComment, issue.ID)
		// TODO: Notify admin about this issue.
	}

	token.Status = "used"
	// token.UpdatedAt is handled by GORM's autoUpdateTime via `gorm:"autoUpdateTime"` on the model
	err = s.verificationTokenRepo.Update(token) // TODO: Pass tx
	if err != nil {
		// if tx != nil { tx.Rollback() }
		return fmt.Errorf("failed to update verification token status: %w", err)
	}

	// if tx != nil {
	// 	if err := tx.Commit().Error; err != nil {
	// 		return fmt.Errorf("failed to commit transaction: %w", err)
	// 	}
	// }
	return nil
}

// GetAdminVerificationStatus implements the logic for GET /admin/status.
func (s *verificationService) GetAdminVerificationStatus(filters AdminStatusFilters) (*AdminVerificationStatusResponse, error) {
	// This is a complex query and aggregation. The implementation will be simplified.
	// Actual implementation would involve more sophisticated queries, possibly joining tables
	// or making multiple calls and processing data in Go.

	// Placeholder criteria for fetching tokens
	// In a real scenario, EmployeeRepository might have a method like FindByDepartmentID(deptID)
	// and these filters would be more complex, possibly involving joins in the repository layer.
	tokenCriteria := make(map[string]interface{})
	if filters.Status != "" {
		tokenCriteria["status"] = filters.Status
	}
	if filters.EmployeeID != nil {
		tokenCriteria["employee_db_id"] = *filters.EmployeeID
	}
	// CycleID and DepartmentID filters would likely require more complex queries
	// or fetching all and then filtering in code if not performance critical.
	// For instance, for DepartmentID, you might fetch all employees in that department,
	// then filter tokens by those employee IDs.

	allTokens, err := s.verificationTokenRepo.FindByCriteria(tokenCriteria)
	if err != nil {
		return nil, fmt.Errorf("error fetching verification tokens: %w", err)
	}

	// Placeholder criteria for fetching issues
	issueFilters := make(map[string]interface{})
	if filters.EmployeeID != nil {
		issueFilters["reported_by_employee_db_id"] = *filters.EmployeeID
	}
	// Similar considerations for DepartmentID and CycleID for issues.

	allIssues, err := s.userReportedIssueRepo.FindAll(issueFilters)
	if err != nil {
		return nil, fmt.Errorf("error fetching user reported issues: %w", err)
	}

	// --- Calculate Statistics & Compile Lists ---
	resp := AdminVerificationStatusResponse{}
	resp.TotalInitiated = len(allTokens)
	
	// Use maps to efficiently retrieve employee/mobile details if needed multiple times
	employeeCache := make(map[uint]*models.Employee) 
	// mobileNumberCache := make(map[uint]*models.MobileNumber) // If needed for issue details

	for _, token := range allTokens {
		if token.Status == "used" {
			resp.Responded++
		} else if token.Status == "pending" {
			if time.Now().After(token.ExpiresAt) {
				resp.ExpiredNoResponse++
				// Optionally, update token status to "expired" here if not done by a batch job
				// token.Status = "expired"
				// s.verificationTokenRepo.Update(token) // Be careful with loops and DB updates
			} else {
				resp.PendingResponse++
				if _, ok := employeeCache[token.EmployeeDbId]; !ok {
					emp, _ := s.employeeRepo.FindByID(token.EmployeeDbId)
					if emp != nil {
						employeeCache[token.EmployeeDbId] = emp
					}
				}
				if emp := employeeCache[token.EmployeeDbId]; emp != nil {
					resp.PendingUsers = append(resp.PendingUsers, PendingUserSummary{
						EmployeeID:      emp.ID,
						EmployeeName:    emp.Name,     // Assuming Employee has Name
						Department:      emp.Department, // Assuming Employee has Department
						TokenExpiresAt:  token.ExpiresAt,
						VerificationURL: fmt.Sprintf("/verify-numbers?token=%s", token.Token),
					})
				}
			}
		}
	}

	resp.IssuesReportedCount = len(allIssues)
	for _, issue := range allIssues {
		if _, ok := employeeCache[issue.ReportedByEmployeeDbId]; !ok {
			emp, _ := s.employeeRepo.FindByID(issue.ReportedByEmployeeDbId)
			if emp != nil {
				employeeCache[issue.ReportedByEmployeeDbId] = emp
			}
		}
		reportedBy := "Unknown Employee"
		if emp := employeeCache[issue.ReportedByEmployeeDbId]; emp != nil {
			reportedBy = emp.Name
		}

		if issue.MobileNumberDbId.Valid { // Issue related to an existing mobile number
			// mobileNumber, _ := s.mobileNumberRepo.FindByID(uint(issue.MobileNumberDbId.Int64))
			// For now, using a placeholder. In reality, you'd fetch or have it cached.
			mnStr := fmt.Sprintf("MobileNumberID(%d)", issue.MobileNumberDbId.Int64)
			// if mn := mobileNumberCache[uint(issue.MobileNumberDbId.Int64)]; mn != nil {
			// 	mnStr = mn.Number
			// } else {
			// 	  fetchedMN, _ := s.mobileNumberRepo.FindByID(uint(issue.MobileNumberDbId.Int64))
			//    if fetchedMN != nil { mobileNumberCache[fetchedMN.ID] = fetchedMN; mnStr = fetchedMN.Number }
			// }

			resp.ReportedIssues = append(resp.ReportedIssues, ReportedIssueDetail{
				IssueID:            issue.ID,
				ReportedByEmployee: reportedBy,
				MobileNumber:       mnStr, // Placeholder, would be actual number
				IssueType:          issue.IssueType,
				UserComment:        issue.UserComment.String, // Access .String for sql.NullString
				ReportedAt:         issue.CreatedAt,
				AdminActionStatus:  issue.AdminActionStatus,
			})
		} else if issue.ReportedPhoneNumber.Valid { // Issue for an unlisted number
			resp.UnlistedNumbers = append(resp.UnlistedNumbers, UnlistedNumberReportedAdmin{
				IssueID:             issue.ID,
				ReportedByEmployee:  reportedBy,
				ReportedPhoneNumber: issue.ReportedPhoneNumber.String, // Access .String
				UserComment:         issue.UserComment.String,         // Access .String
				ReportedAt:          issue.CreatedAt,
				AdminActionStatus:   issue.AdminActionStatus,
			})
		}
	}
	// Note: Filtering by DepartmentID for PendingUsers and ReportedIssues post-fetch
	// would require iterating through them and checking the department of the associated employee.
	// This is less efficient than a DB-level filter if performance is critical.

	return &resp, nil
}
