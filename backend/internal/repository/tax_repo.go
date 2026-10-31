package repository

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

type TaxRepo struct{}

func NewTaxRepo() *TaxRepo { return &TaxRepo{} }

func (r *TaxRepo) Create(db *gorm.DB, t *models.Tax) error { return db.Create(t).Error }

func (r *TaxRepo) Update(db *gorm.DB, id int64, fields map[string]any) error {
	return db.Model(&models.Tax{}).Where("id = ?", id).Updates(fields).Error
}

// FindByID returns (nil, nil) when missing.
func (r *TaxRepo) FindByID(db *gorm.DB, id int64) (*models.Tax, error) {
	var t models.Tax
	err := db.First(&t, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &t, err
}

// NameTaken compares names ignoring case, matching the ux_taxes_name_ci index.
func (r *TaxRepo) NameTaken(db *gorm.DB, name string, exceptID int64) (bool, error) {
	var n int64
	err := db.Model(&models.Tax{}).Where("lower(name) = lower(?) AND id <> ?", strings.TrimSpace(name), exceptID).Count(&n).Error
	return n > 0, err
}

type TaxFilter struct {
	Search string
	Active *bool
}

func (r *TaxRepo) List(db *gorm.DB, f TaxFilter, p pagination.Params) ([]models.Tax, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		if s := strings.TrimSpace(f.Search); s != "" {
			q = q.Where("lower(name) LIKE ?", "%"+escapeLike(strings.ToLower(s))+"%")
		}
		if f.Active != nil {
			q = q.Where("is_active = ?", *f.Active)
		}
		return q
	}
	var total int64
	if err := db.Model(&models.Tax{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []models.Tax
	err := db.Scopes(scope).Order(p.Sort).Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}

// ProductsUsing counts ACTIVE, non-archived products whose default tax is each id.
// One grouped query for a whole page of taxes (no N+1).
func (r *TaxRepo) ProductsUsing(db *gorm.DB, taxIDs []int64) (map[int64]int64, error) {
	out := make(map[int64]int64, len(taxIDs))
	if len(taxIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		TaxID int64
		N     int64
	}
	err := db.Model(&models.Product{}).
		Select("tax_id, count(*) AS n").
		Where("tax_id IN ? AND status = ?", taxIDs, models.StatusActive).
		Group("tax_id").Scan(&rows).Error
	for _, r := range rows {
		out[r.TaxID] = r.N
	}
	return out, err
}
