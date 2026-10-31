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

func (r *CustomerRepo) Update(db *gorm.DB, id int64, fields map[string]any) error {
	return db.Model(&models.Customer{}).Where("id = ?", id).Updates(fields).Error
}
