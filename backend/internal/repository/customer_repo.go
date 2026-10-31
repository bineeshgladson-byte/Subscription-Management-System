package repository

import (
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

type CustomerRepo struct{}

func NewCustomerRepo() *CustomerRepo { return &CustomerRepo{} }

func (r *CustomerRepo) Create(db *gorm.DB, c *models.Customer) error {
	return db.Create(c).Error
}
