package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

// TaxService: "Taxes are calculated automatically from rules the admin has set."
// Admin manages taxes; staff can read them (to show on quotations/invoices).
//
// Taxes are never hard-deleted: invoice lines keep their own copy of the rate
// (tax_rate), so history never changes, and DELETE only deactivates.
type TaxService struct {
	db    *gorm.DB
	taxes *repository.TaxRepo
}

func NewTaxService(db *gorm.DB, taxes *repository.TaxRepo) *TaxService {
	return &TaxService{db: db, taxes: taxes}
}

func (s *TaxService) List(ctx context.Context, f repository.TaxFilter, p pagination.Params) ([]dto.TaxResponse, int64, error) {
	db := s.db.WithContext(ctx)
	list, total, err := s.taxes.List(db, f, p)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]int64, len(list))
	for i := range list {
		ids[i] = list[i].ID
	}
	using, err := s.taxes.ProductsUsing(db, ids)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.TaxResponse, len(list))
	for i := range list {
		out[i] = dto.NewTaxResponse(&list[i], using[list[i].ID])
	}
	return out, total, nil
}

func (s *TaxService) Get(ctx context.Context, id int64) (*dto.TaxResponse, error) {
	db := s.db.WithContext(ctx)
	t, err := s.taxes.FindByID(db, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, apperr.NotFound("tax")
	}
	using, err := s.taxes.ProductsUsing(db, []int64{id})
	if err != nil {
		return nil, err
	}
	out := dto.NewTaxResponse(t, using[id])
	return &out, nil
}

func (s *TaxService) Create(ctx context.Context, in dto.CreateTaxRequest) (*dto.TaxResponse, error) {
	name := strings.TrimSpace(in.Name)
	rate, reason := parseRate(in.Rate)
	if reason != "" {
		return nil, apperr.Validation("Some fields are invalid", map[string]string{"rate": reason})
	}
	db := s.db.WithContext(ctx)
	if taken, err := s.taxes.NameTaken(db, name, 0); err != nil {
		return nil, err
	} else if taken {
		return nil, taxNameTaken()
	}
	t := &models.Tax{Name: name, Rate: rate, IsActive: true}
	if err := s.taxes.Create(db, t); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, taxNameTaken()
		}
		return nil, err
	}
	return s.Get(ctx, t.ID)
}

// Update changes name/rate and can reactivate or deactivate. A new rate only
// affects invoices created from now on.
func (s *TaxService) Update(ctx context.Context, id int64, in dto.UpdateTaxRequest) (*dto.TaxResponse, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	db := s.db.WithContext(ctx)
	fields := map[string]any{}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if taken, err := s.taxes.NameTaken(db, name, id); err != nil {
			return nil, err
		} else if taken {
			return nil, taxNameTaken()
		}
		fields["name"] = name
	}
	if in.Rate != nil {
		rate, reason := parseRate(*in.Rate)
		if reason != "" {
			return nil, apperr.Validation("Some fields are invalid", map[string]string{"rate": reason})
		}
		fields["rate"] = rate
	}
	if in.IsActive != nil {
		if !*in.IsActive && current.IsActive && current.ProductsUsing > 0 {
			return nil, taxInUse(current.ProductsUsing)
		}
		fields["is_active"] = *in.IsActive
	}
	if len(fields) > 0 {
		if err := s.taxes.Update(db, id, fields); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, taxNameTaken()
			}
			return nil, err
		}
	}
	return s.Get(ctx, id)
}

// Deactivate is what DELETE /taxes/:id does. Refused while active products
// use the tax as their default, so new quotations never pick up a switched-off tax.
func (s *TaxService) Deactivate(ctx context.Context, id int64) (*dto.TaxResponse, error) {
	off := false
	return s.Update(ctx, id, dto.UpdateTaxRequest{IsActive: &off})
}

// parseRate accepts "18", "18.0" or "18.00": 0 to 100, at most 2 decimals.
func parseRate(raw string) (decimal.Decimal, string) {
	d, err := decimal.NewFromString(strings.TrimSpace(raw))
	switch {
	case err != nil:
		return d, "must be a number, e.g. \"18.00\""
	case d.IsNegative() || d.GreaterThan(decimal.NewFromInt(100)):
		return d, "must be between 0 and 100"
	case d.Exponent() < -2 && !d.Equal(d.Round(2)):
		return d, "can have at most 2 decimal places"
	}
	return d.Round(2), ""
}

func taxNameTaken() error {
	e := apperr.Conflict("a tax with this name already exists")
	e.Details = map[string]string{"name": "is already used by another tax"}
	return e
}

func taxInUse(n int64) error {
	e := apperr.Rule("TAX_IN_USE", fmt.Sprintf(
		"%d active product(s) use this tax as their default; give them another tax first", n))
	e.Details["products_using"] = fmt.Sprint(n)
	return e
}
