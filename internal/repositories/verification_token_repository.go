package repositories

import (
	"errors"

	"gorm.io/gorm"
	"navigator/internal/models"
)

// VerificationTokenRepository defines the interface for verification token database operations.
type VerificationTokenRepository interface {
	Create(token *models.VerificationToken) error
	FindByToken(tokenStr string) (*models.VerificationToken, error)
	Update(token *models.VerificationToken) error
	FindPendingByEmployeeID(employeeID uint) ([]*models.VerificationToken, error)
	FindByCriteria(criteria map[string]interface{}) ([]*models.VerificationToken, error)
}

type verificationTokenRepository struct {
	db *gorm.DB
}

// NewVerificationTokenRepository creates a new instance of VerificationTokenRepository.
func NewVerificationTokenRepository(db *gorm.DB) VerificationTokenRepository {
	return &verificationTokenRepository{db: db}
}

// Create inserts a new VerificationToken into the database.
func (r *verificationTokenRepository) Create(token *models.VerificationToken) error {
	return r.db.Create(token).Error
}

// FindByToken retrieves a VerificationToken by its token string.
func (r *verificationTokenRepository) FindByToken(tokenStr string) (*models.VerificationToken, error) {
	var token models.VerificationToken
	err := r.db.Where("token = ?", tokenStr).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Or return a specific "not found" error
		}
		return nil, err
	}
	return &token, nil
}

// Update modifies an existing VerificationToken in the database.
func (r *verificationTokenRepository) Update(token *models.VerificationToken) error {
	return r.db.Save(token).Error
}

// FindPendingByEmployeeID retrieves all pending tokens for a given employee.
func (r *verificationTokenRepository) FindPendingByEmployeeID(employeeID uint) ([]*models.VerificationToken, error) {
	var tokens []*models.VerificationToken
	err := r.db.Where("employee_db_id = ? AND status = ?", employeeID, "pending").Find(&tokens).Error
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// FindByCriteria retrieves verification tokens based on a map of criteria.
func (r *verificationTokenRepository) FindByCriteria(criteria map[string]interface{}) ([]*models.VerificationToken, error) {
	var tokens []*models.VerificationToken
	err := r.db.Where(criteria).Find(&tokens).Error
	if err != nil {
		return nil, err
	}
	return tokens, nil
}
