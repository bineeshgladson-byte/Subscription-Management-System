package repository

import (
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

type CustomerRepo struct{}

func NewCustomerRepo() *CustomerRepo { return &CustomerRepo{} }

func (r *CustomerRepo) Create(db *gorm.DB, c *models.Customer) error {
	return db.Create(c).Error
}

func (r *CustomerRepo) Update(db *gorm.DB, id int64, fields map[string]any) error {
	return db.Model(&models.Customer{}).Where("id = ?", id).Updates(fields).Error
}

// FindByID returns (nil, nil) if missing or archived.
func (r *CustomerRepo) FindByID(db *gorm.DB, id int64) (*models.Customer, error) {
	var c models.Customer
	err := db.First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

// FindLiveByEmail finds the non-archived customer with this email (case-insensitive).
func (r *CustomerRepo) FindLiveByEmail(db *gorm.DB, email string) (*models.Customer, error) {
	var c models.Customer
	err := db.Where("lower(email) = lower(?)", email).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &c, err
}

// EmailTaken: is another live customer already using this email?
func (r *CustomerRepo) EmailTaken(db *gorm.DB, email string, exceptID int64) (bool, error) {
	var n int64
	err := db.Model(&models.Customer{}).
		Where("lower(email) = lower(?) AND id <> ?", email, exceptID).Count(&n).Error
	return n > 0, err
}

type CustomerFilter struct {
	Search   string // name, email, company or phone
	HasLogin *bool
}

func (r *CustomerRepo) List(db *gorm.DB, f CustomerFilter, p pagination.Params) ([]models.Customer, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		if s := strings.TrimSpace(f.Search); s != "" {
			like := "%" + escapeLike(strings.ToLower(s)) + "%"
			q = q.Where("(lower(name) LIKE ? OR lower(email) LIKE ? OR lower(coalesce(company_name,'')) LIKE ? OR coalesce(phone,'') LIKE ?)",
				like, like, like, like)
		}
		if f.HasLogin != nil {
			if *f.HasLogin {
				q = q.Where("user_id IS NOT NULL")
			} else {
				q = q.Where("user_id IS NULL")
			}
		}
		return q
	}
	var total int64
	if err := db.Model(&models.Customer{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []models.Customer
	err := db.Scopes(scope).Order(p.Sort).Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}

type CustomerStats struct {
	ActiveSubscriptions int64
	TotalSubscriptions  int64
	LiveSubscriptions   int64 // draft, quotation, confirmed, active or paused: blocks archiving
	OpenInvoices        int64
	OverdueInvoices     int64
	AmountDue           decimal.Decimal
}

// Stats runs one round trip; every sub-query uses the customer_id indexes.
func (r *CustomerRepo) Stats(db *gorm.DB, customerID int64, today time.Time) (CustomerStats, error) {
	var row struct {
		Active, Total, Live, Open, Overdue int64
		Due                                decimal.Decimal
	}
	err := db.Raw(`
		SELECT
		  (SELECT count(*) FROM subscriptions WHERE customer_id = @id AND status IN ('active','paused'))      AS active,
		  (SELECT count(*) FROM subscriptions WHERE customer_id = @id)                                      AS total,
		  (SELECT count(*) FROM subscriptions WHERE customer_id = @id
		       AND status IN ('draft','quotation','confirmed','active','paused'))                           AS live,
		  (SELECT count(*) FROM invoices WHERE customer_id = @id AND status = 'confirmed' AND amount_due > 0) AS open,
		  (SELECT count(*) FROM invoices WHERE customer_id = @id AND status = 'confirmed' AND amount_due > 0
		       AND due_date < @today)                                                                       AS overdue,
		  (SELECT coalesce(sum(amount_due), 0) FROM invoices WHERE customer_id = @id AND status = 'confirmed') AS due`,
		map[string]any{"id": customerID, "today": today.Format("2006-01-02")}).Scan(&row).Error
	return CustomerStats{ActiveSubscriptions: row.Active, TotalSubscriptions: row.Total, LiveSubscriptions: row.Live,
		OpenInvoices: row.Open, OverdueInvoices: row.Overdue, AmountDue: row.Due}, err
}

// Archive soft-deletes the customer (sets deleted_at). Invoices keep pointing at it.
func (r *CustomerRepo) Archive(db *gorm.DB, id int64) error {
	return db.Delete(&models.Customer{}, id).Error
}
