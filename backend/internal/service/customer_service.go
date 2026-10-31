package service

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/internal/validator"
	"github.com/brightloop/subscriptions/backend/pkg/clock"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

// CustomerService: "a small internal team handles customers day to day".
// Admin and staff can list, create and edit; only the admin can archive.
type CustomerService struct {
	db        *gorm.DB
	customers *repository.CustomerRepo
	users     *repository.UserRepo
	clock     clock.Clock
}

func NewCustomerService(db *gorm.DB, customers *repository.CustomerRepo, users *repository.UserRepo, clk clock.Clock) *CustomerService {
	return &CustomerService{db: db, customers: customers, users: users, clock: clk}
}

func (s *CustomerService) List(ctx context.Context, f repository.CustomerFilter, p pagination.Params) ([]models.Customer, int64, error) {
	return s.customers.List(s.db.WithContext(ctx), f, p)
}

func (s *CustomerService) Get(ctx context.Context, id int64) (*models.Customer, error) {
	c, err := s.customers.FindByID(s.db.WithContext(ctx), id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperr.NotFound("customer")
	}
	return c, nil
}

// Detail returns the customer plus subscription and invoice counts.
func (s *CustomerService) Detail(ctx context.Context, id int64) (*dto.CustomerDetailResponse, error) {
	c, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	st, err := s.customers.Stats(s.db.WithContext(ctx), id, s.clock.Today())
	if err != nil {
		return nil, err
	}
	return &dto.CustomerDetailResponse{
		CustomerResponse: dto.NewCustomerResponse(c),
		Stats: dto.CustomerStats{ActiveSubscriptions: st.ActiveSubscriptions, TotalSubscriptions: st.TotalSubscriptions,
			OpenInvoices: st.OpenInvoices, OverdueInvoices: st.OverdueInvoices, AmountDue: st.AmountDue.StringFixed(2)},
	}, nil
}

func (s *CustomerService) Create(ctx context.Context, actorID int64, in dto.CreateCustomerRequest) (*models.Customer, error) {
	errs := map[string]string{}
	email := validator.NormalizeEmail(in.Email)
	if email == "" {
		errs["email"] = "is not a valid email"
	}
	phone, company, address := optional(in.Phone), optional(in.CompanyName), optional(in.BillingAddress)
	if phone != nil && validator.Phone(*phone) == "" {
		errs["phone"] = "is not a valid phone number"
	}
	var gstin *string
	if g := optional(in.GSTIN); g != nil {
		if v := validator.GSTIN(*g); v == "" {
			errs["gstin"] = "is not a valid GSTIN"
		} else {
			gstin = &v
		}
	}
	if len(errs) > 0 {
		return nil, apperr.Validation("Some fields are invalid", errs)
	}

	db := s.db.WithContext(ctx)
	taken, err := s.customers.EmailTaken(db, email, 0)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, customerEmailTaken()
	}

	c := &models.Customer{Name: in.Name, Email: email, Phone: phone, CompanyName: company,
		BillingAddress: address, GSTIN: gstin, CreatedBy: &actorID}
	if err := s.customers.Create(db, c); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, customerEmailTaken()
		}
		return nil, err
	}
	return c, nil
}

func (s *CustomerService) Update(ctx context.Context, id int64, in dto.UpdateCustomerRequest) (*models.Customer, error) {
	c, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	fields := map[string]any{}
	errs := map[string]string{}
	if in.Name != nil {
		fields["name"] = *in.Name
	}
	if in.Email != nil {
		email := validator.NormalizeEmail(*in.Email)
		switch {
		case email == "":
			errs["email"] = "is not a valid email"
		case c.UserID != nil && email != c.Email:
			// Their email is also their login. Changing it here would lock them out.
			errs["email"] = "can't be changed: this customer logs in with this email"
		case email != c.Email:
			taken, err := s.customers.EmailTaken(s.db.WithContext(ctx), email, c.ID)
			if err != nil {
				return nil, err
			}
			if taken {
				return nil, customerEmailTaken()
			}
			fields["email"] = email
		}
	}
	if in.Phone != nil {
		if *in.Phone != "" && validator.Phone(*in.Phone) == "" {
			errs["phone"] = "is not a valid phone number"
		} else {
			fields["phone"] = nullIfEmpty(*in.Phone)
		}
	}
	if in.CompanyName != nil {
		fields["company_name"] = nullIfEmpty(*in.CompanyName)
	}
	if in.BillingAddress != nil {
		fields["billing_address"] = nullIfEmpty(*in.BillingAddress)
	}
	if in.GSTIN != nil {
		if *in.GSTIN == "" {
			fields["gstin"] = nil
		} else if g := validator.GSTIN(*in.GSTIN); g == "" {
			errs["gstin"] = "is not a valid GSTIN"
		} else {
			fields["gstin"] = g
		}
	}
	if len(errs) > 0 {
		return nil, apperr.Validation("Some fields are invalid", errs)
	}
	if len(fields) == 0 {
		return c, nil
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.customers.Update(tx, c.ID, fields); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return customerEmailTaken()
			}
			return err
		}
		// Keep the login's display name in sync with the customer record.
		if c.UserID != nil && in.Name != nil {
			return s.users.Update(tx, *c.UserID, map[string]any{"full_name": *in.Name})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Archive hides a customer from lists and blocks their login. It's refused
// while they still have live subscriptions/quotations or unpaid invoices, so
// nothing billable is orphaned. Their invoices and payments are kept.
func (s *CustomerService) Archive(ctx context.Context, id int64) error {
	c, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	st, err := s.customers.Stats(s.db.WithContext(ctx), id, s.clock.Today())
	if err != nil {
		return err
	}
	if st.LiveSubscriptions > 0 || st.OpenInvoices > 0 {
		e := apperr.Rule("CUSTOMER_HAS_OPEN_ITEMS", fmt.Sprintf(
			"close this customer's %d open subscription(s)/quotation(s) and settle %d unpaid invoice(s) first",
			st.LiveSubscriptions, st.OpenInvoices))
		e.Details["live_subscriptions"] = fmt.Sprint(st.LiveSubscriptions)
		e.Details["open_invoices"] = fmt.Sprint(st.OpenInvoices)
		return e
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.customers.Archive(tx, c.ID); err != nil {
			return err
		}
		if c.UserID != nil {
			return s.users.Update(tx, *c.UserID, map[string]any{"is_active": false})
		}
		return nil
	})
}

func customerEmailTaken() error {
	e := apperr.Conflict("a customer with this email already exists")
	e.Details = map[string]string{"email": "is already used by another customer"}
	return e
}

// optional turns "" into nil so empty strings are stored as NULL.
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
