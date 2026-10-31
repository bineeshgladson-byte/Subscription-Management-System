package repository

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

type ProductRepo struct{}

func NewProductRepo() *ProductRepo { return &ProductRepo{} }

// withDetails preloads the tax and variants (2 extra queries per page, not per row).
func withDetails(db *gorm.DB) *gorm.DB {
	return db.Preload("Tax").Preload("Variants", func(q *gorm.DB) *gorm.DB {
		return q.Order("attribute, name")
	})
}

func (r *ProductRepo) Create(db *gorm.DB, p *models.Product) error {
	return db.Omit("Tax", "Variants").Create(p).Error
}

func (r *ProductRepo) Update(db *gorm.DB, id int64, fields map[string]any) error {
	return db.Model(&models.Product{}).Where("id = ?", id).Updates(fields).Error
}

// FindByID returns (nil, nil) if missing. Includes archived products.
func (r *ProductRepo) FindByID(db *gorm.DB, id int64) (*models.Product, error) {
	var p models.Product
	err := withDetails(db).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

// FindByIDs loads several products with variants (used to validate plan lines).
func (r *ProductRepo) FindByIDs(db *gorm.DB, ids []int64) (map[int64]*models.Product, error) {
	var list []models.Product
	if err := withDetails(db).Where("id IN ?", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	out := make(map[int64]*models.Product, len(list))
	for i := range list {
		out[list[i].ID] = &list[i]
	}
	return out, nil
}

// ActiveNameTaken: is another ACTIVE product using this name (ignoring case)?
func (r *ProductRepo) ActiveNameTaken(db *gorm.DB, name string, exceptID int64) (bool, error) {
	var n int64
	err := db.Model(&models.Product{}).
		Where("lower(name) = lower(?) AND status = ? AND id <> ?", strings.TrimSpace(name), models.StatusActive, exceptID).
		Count(&n).Error
	return n > 0, err
}

type ProductFilter struct {
	Search      string
	Status      string // "active" (default), "archived" or "all"
	ProductType string
	TaxID       int64
}

func (r *ProductRepo) List(db *gorm.DB, f ProductFilter, p pagination.Params) ([]models.Product, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		if s := strings.TrimSpace(f.Search); s != "" {
			like := "%" + escapeLike(strings.ToLower(s)) + "%"
			q = q.Where("(lower(name) LIKE ? OR lower(coalesce(description,'')) LIKE ?)", like, like)
		}
		switch f.Status {
		case "all":
		case "archived":
			q = q.Where("status = ?", models.StatusArchived)
		default:
			q = q.Where("status = ?", models.StatusActive)
		}
		if f.ProductType != "" {
			q = q.Where("product_type = ?", f.ProductType)
		}
		if f.TaxID > 0 {
			q = q.Where("tax_id = ?", f.TaxID)
		}
		return q
	}
	var total int64
	if err := db.Model(&models.Product{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []models.Product
	err := withDetails(db).Scopes(scope).Order(p.Sort).Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}

// ActivePlansUsing counts ACTIVE plans that include this product (or, if
// variantID > 0, this specific variant).
func (r *ProductRepo) ActivePlansUsing(db *gorm.DB, productID, variantID int64) (int64, error) {
	var n int64
	q := db.Table("plan_products pp").
		Joins("JOIN subscription_plans sp ON sp.id = pp.plan_id AND sp.status = 'active'").
		Where("pp.product_id = ?", productID)
	if variantID > 0 {
		q = q.Where("pp.variant_id = ?", variantID)
	}
	err := q.Distinct("pp.plan_id").Count(&n).Error
	return n, err
}

// ---------- variants ----------

func (r *ProductRepo) CreateVariant(db *gorm.DB, v *models.ProductVariant) error {
	return db.Create(v).Error
}

func (r *ProductRepo) UpdateVariant(db *gorm.DB, id int64, fields map[string]any) error {
	return db.Model(&models.ProductVariant{}).Where("id = ?", id).Updates(fields).Error
}

// FindVariant returns the variant only if it belongs to productID; (nil, nil) otherwise.
func (r *ProductRepo) FindVariant(db *gorm.DB, productID, variantID int64) (*models.ProductVariant, error) {
	var v models.ProductVariant
	err := db.Where("product_id = ?", productID).First(&v, variantID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &v, err
}

func (r *ProductRepo) VariantNameTaken(db *gorm.DB, productID int64, attribute, name string, exceptID int64) (bool, error) {
	var n int64
	err := db.Model(&models.ProductVariant{}).
		Where("product_id = ? AND lower(attribute) = lower(?) AND lower(name) = lower(?) AND id <> ?",
			productID, strings.TrimSpace(attribute), strings.TrimSpace(name), exceptID).Count(&n).Error
	return n > 0, err
}

func (r *ProductRepo) SKUTaken(db *gorm.DB, sku string, exceptID int64) (bool, error) {
	var n int64
	err := db.Model(&models.ProductVariant{}).Where("sku = ? AND id <> ?", sku, exceptID).Count(&n).Error
	return n > 0, err
}
