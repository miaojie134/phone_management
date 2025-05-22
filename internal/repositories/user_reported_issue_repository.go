package repositories

import (
	"errors"

	"gorm.io/gorm"
	"navigator/internal/models"
)

// UserReportedIssueRepository defines the interface for user reported issue database operations.
type UserReportedIssueRepository interface {
	Create(issue *models.UserReportedIssue) error
	FindByVerificationTokenID(tokenID uint) ([]*models.UserReportedIssue, error)
	FindAll(filters map[string]interface{}) ([]*models.UserReportedIssue, error) // Added filters
	Update(issue *models.UserReportedIssue) error
	FindByID(id uint) (*models.UserReportedIssue, error) // Added for potential direct lookup
}

type userReportedIssueRepository struct {
	db *gorm.DB
}

// NewUserReportedIssueRepository creates a new instance of UserReportedIssueRepository.
func NewUserReportedIssueRepository(db *gorm.DB) UserReportedIssueRepository {
	return &userReportedIssueRepository{db: db}
}

// Create inserts a new UserReportedIssue into the database.
func (r *userReportedIssueRepository) Create(issue *models.UserReportedIssue) error {
	return r.db.Create(issue).Error
}

// FindByVerificationTokenID retrieves UserReportedIssues by their verification token ID.
// Note: VerificationTokenId is sql.NullInt64, so we handle its potential nullness if needed,
// but here we assume we are looking for non-null token IDs.
func (r *userReportedIssueRepository) FindByVerificationTokenID(tokenID uint) ([]*models.UserReportedIssue, error) {
	var issues []*models.UserReportedIssue
	// Assuming VerificationTokenId in the DB model is named verification_token_id
	err := r.db.Where("verification_token_id = ?", tokenID).Find(&issues).Error
	if err != nil {
		return nil, err
	}
	return issues, nil
}

// FindAll retrieves all UserReportedIssues, potentially with filters.
func (r *userReportedIssueRepository) FindAll(filters map[string]interface{}) ([]*models.UserReportedIssue, error) {
	var issues []*models.UserReportedIssue
	query := r.db
	if filters != nil {
		query = query.Where(filters)
	}
	err := query.Find(&issues).Error
	if err != nil {
		return nil, err
	}
	return issues, nil
}

// Update modifies an existing UserReportedIssue in the database.
func (r *userReportedIssueRepository) Update(issue *models.UserReportedIssue) error {
	return r.db.Save(issue).Error
}

// FindByID retrieves a UserReportedIssue by its ID.
func (r *userReportedIssueRepository) FindByID(id uint) (*models.UserReportedIssue, error) {
	var issue models.UserReportedIssue
	err := r.db.First(&issue, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Or return a specific "not found" error
		}
		return nil, err
	}
	return &issue, nil
}
