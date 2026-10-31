package service

import (
	"context"
	"errors"
	"strings"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

// TemplateService — A5. Brief: "they start from a quotation template that
// already has the right plan and product lines, and stays valid for a set
// number of days." Admin writes; admin + staff read. Never hard-deleted.
type TemplateService struct {
	db        *gorm.DB
	templates *repository.TemplateRepo
	plans     *repository.PlanRepo
	products  *repository.ProductRepo
}

func NewTemplateService(db *gorm.DB, templates *repository.TemplateRepo, plans *repository.PlanRepo, products *repository.ProductRepo) *TemplateService {
	return &TemplateService{db: db, templates: templates, plans: plans, products: products}
}

func (s *TemplateService) List(ctx context.Context, f repository.TemplateFilter, p pagination.Params) ([]dto.TemplateResponse, int64, error) {
	list, total, err := s.templates.List(s.db.WithContext(ctx), f, p)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.TemplateResponse, len(list))
	for i := range list {
		out[i] = templateResponse(&list[i])
	}
	return out, total, nil
}

func (s *TemplateService) Get(ctx context.Context, id int64) (*dto.TemplateResponse, error) {
	t, err := s.templates.FindByID(s.db.WithContext(ctx), id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, apperr.NotFound("quotation template")
	}
	out := templateResponse(t)
	return &out, nil
}

func (s *TemplateService) Create(ctx context.Context, in dto.CreateTemplateRequest) (*dto.TemplateResponse, error) {
	db := s.db.WithContext(ctx)
	name := strings.TrimSpace(in.Name)
	if taken, err := s.templates.NameTaken(db, name, 0); err != nil {
		return nil, err
	} else if taken {
		return nil, templateNameTaken()
	}
	plan, err := s.activePlan(db, in.PlanID)
	if err != nil {
		return nil, err
	}
	specs := templateSpecs(in.Lines)
	if len(specs) == 0 {
		if plan == nil {
			return nil, apperr.Validation("Some fields are invalid",
				map[string]string{"lines": "add at least one product line, or choose a plan to copy its products"})
		}
		for _, pp := range plan.Products {
			specs = append(specs, lineSpec{ProductID: pp.ProductID, VariantID: pp.VariantID, Quantity: pp.Quantity})
		}
	}
	lines, err := s.buildLines(db, specs)
	if err != nil {
		return nil, err
	}
	terms := 15
	if in.PaymentTermsDays != nil {
		terms = *in.PaymentTermsDays
	}
	t := &models.QuotationTemplate{Name: name, PlanID: in.PlanID, ValidityDays: defaultInt(in.ValidityDays, 15),
		PaymentTermsDays: terms, Terms: optional(strings.TrimSpace(in.Terms)), IsActive: true}
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := s.templates.Create(tx, t); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return templateNameTaken()
			}
			return err
		}
		return s.templates.ReplaceLines(tx, t.ID, lines)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, t.ID)
}

func (s *TemplateService) Update(ctx context.Context, id int64, in dto.UpdateTemplateRequest) (*dto.TemplateResponse, error) {
	db := s.db.WithContext(ctx)
	t, err := s.templates.FindByID(db, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, apperr.NotFound("quotation template")
	}
	fields := map[string]any{}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if taken, err := s.templates.NameTaken(db, name, id); err != nil {
			return nil, err
		} else if taken {
			return nil, templateNameTaken()
		}
		fields["name"] = name
	}
	if in.PlanID != nil {
		if *in.PlanID == 0 {
			fields["plan_id"] = nil
		} else {
			if _, err := s.activePlan(db, in.PlanID); err != nil {
				return nil, err
			}
			fields["plan_id"] = *in.PlanID
		}
	}
	if in.ValidityDays != nil {
		fields["validity_days"] = *in.ValidityDays
	}
	if in.PaymentTermsDays != nil {
		fields["payment_terms_days"] = *in.PaymentTermsDays
	}
	if in.Terms != nil {
		fields["terms"] = nullIfEmpty(strings.TrimSpace(*in.Terms))
	}
	if in.IsActive != nil {
		fields["is_active"] = *in.IsActive
	}
	var lines []models.QuotationTemplateLine
	if in.Lines != nil {
		if lines, err = s.buildLines(db, templateSpecs(in.Lines)); err != nil {
			return nil, err
		}
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if len(fields) > 0 {
			if err := s.templates.Update(tx, id, fields); err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					return templateNameTaken()
				}
				return err
			}
		}
		if in.Lines != nil {
			return s.templates.ReplaceLines(tx, id, lines)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Deactivate is DELETE /quotation-templates/:id.
func (s *TemplateService) Deactivate(ctx context.Context, id int64) (*dto.TemplateResponse, error) {
	off := false
	return s.Update(ctx, id, dto.UpdateTemplateRequest{IsActive: &off})
}

func (s *TemplateService) activePlan(db *gorm.DB, planID *int64) (*models.SubscriptionPlan, error) {
	if planID == nil {
		return nil, nil
	}
	p, err := s.plans.FindByID(db, *planID)
	if err != nil {
		return nil, err
	}
	if p == nil || p.Status != models.StatusActive {
		return nil, apperr.Validation("Some fields are invalid", map[string]string{"plan_id": "must be an active plan"})
	}
	return p, nil
}

func (s *TemplateService) buildLines(db *gorm.DB, specs []lineSpec) ([]models.QuotationTemplateLine, error) {
	resolved, err := resolveLines(db, s.products, "lines", specs, false)
	if err != nil {
		return nil, err
	}
	out := make([]models.QuotationTemplateLine, len(resolved))
	for i, r := range resolved {
		out[i] = models.QuotationTemplateLine{ProductID: r.ProductID, VariantID: r.VariantID, Quantity: r.Quantity}
	}
	return out, nil
}

func templateSpecs(in []dto.TemplateLineInput) []lineSpec {
	out := make([]lineSpec, len(in))
	for i, l := range in {
		out[i] = lineSpec{ProductID: l.ProductID, VariantID: l.VariantID, Quantity: l.Quantity}
	}
	return out
}

func templateResponse(t *models.QuotationTemplate) dto.TemplateResponse {
	r := dto.TemplateResponse{ID: t.ID, Name: t.Name, ValidityDays: t.ValidityDays, PaymentTermsDays: t.PaymentTermsDays,
		Terms: t.Terms, IsActive: t.IsActive, Lines: make([]dto.TemplateLineResponse, 0, len(t.Lines)),
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
	if t.Plan != nil {
		r.Plan = &dto.PlanSummary{ID: t.Plan.ID, Name: t.Plan.Name, Price: t.Plan.Price.StringFixed(2),
			BillingLabel: billingLabel(t.Plan.BillingPeriod, t.Plan.BillingInterval), Status: string(t.Plan.Status)}
	}
	total := decimal.Zero
	for _, l := range t.Lines {
		lr := dto.TemplateLineResponse{ID: l.ID, ProductID: l.ProductID, VariantID: l.VariantID, Quantity: l.Quantity}
		unit := decimal.Zero
		if l.Product != nil {
			lr.ProductName = l.Product.Name
			unit = l.Product.SalesPrice
		}
		if l.Variant != nil {
			n := l.Variant.Name
			lr.VariantName = &n
			unit = unit.Add(l.Variant.ExtraPrice)
		}
		line := unit.Mul(decimal.NewFromInt(int64(l.Quantity)))
		total = total.Add(line)
		lr.UnitPrice, lr.LineTotal = unit.StringFixed(2), line.StringFixed(2)
		r.Lines = append(r.Lines, lr)
	}
	r.EstimatedTotal = total.StringFixed(2)
	return r
}

func templateNameTaken() error {
	e := apperr.Conflict("a quotation template with this name already exists")
	e.Details = map[string]string{"name": "is already used by another template"}
	return e
}
