package repository

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

type TemplateRepo struct{}

func NewTemplateRepo() *TemplateRepo { return &TemplateRepo{} }

func withTemplateDetails(db *gorm.DB) *gorm.DB {
	return db.Preload("Plan").
		Preload("Lines", func(q *gorm.DB) *gorm.DB { return q.Order("id") }).
		Preload("Lines.Product").Preload("Lines.Variant")
}

func (r *TemplateRepo) Create(db *gorm.DB, t *models.QuotationTemplate) error {
	return db.Omit("Plan", "Lines").Create(t).Error
}

func (r *TemplateRepo) Update(db *gorm.DB, id int64, fields map[string]any) error {
	return db.Model(&models.QuotationTemplate{}).Where("id = ?", id).Updates(fields).Error
}

func (r *TemplateRepo) FindByID(db *gorm.DB, id int64) (*models.QuotationTemplate, error) {
	var t models.QuotationTemplate
	err := withTemplateDetails(db).First(&t, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &t, err
}

func (r *TemplateRepo) NameTaken(db *gorm.DB, name string, exceptID int64) (bool, error) {
	var n int64
	err := db.Model(&models.QuotationTemplate{}).
		Where("lower(name) = lower(?) AND id <> ?", strings.TrimSpace(name), exceptID).Count(&n).Error
	return n > 0, err
}

func (r *TemplateRepo) ReplaceLines(tx *gorm.DB, templateID int64, lines []models.QuotationTemplateLine) error {
	if err := tx.Where("template_id = ?", templateID).Delete(&models.QuotationTemplateLine{}).Error; err != nil {
		return err
	}
	for i := range lines {
		lines[i].ID, lines[i].TemplateID = 0, templateID
	}
	if len(lines) == 0 {
		return nil
	}
	return tx.Omit("Product", "Variant").Create(&lines).Error
}

type TemplateFilter struct {
	Search string
	Active *bool
	PlanID int64
}

func (r *TemplateRepo) List(db *gorm.DB, f TemplateFilter, p pagination.Params) ([]models.QuotationTemplate, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		if s := strings.TrimSpace(f.Search); s != "" {
			q = q.Where("lower(name) LIKE ?", "%"+escapeLike(strings.ToLower(s))+"%")
		}
		if f.Active != nil {
			q = q.Where("is_active = ?", *f.Active)
		}
		if f.PlanID > 0 {
			q = q.Where("plan_id = ?", f.PlanID)
		}
		return q
	}
	var total int64
	if err := db.Model(&models.QuotationTemplate{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []models.QuotationTemplate
	err := withTemplateDetails(db).Scopes(scope).Order(p.Sort).Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}
