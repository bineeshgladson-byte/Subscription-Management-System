// Package hash wraps bcrypt so the cost is configured in one place.
package hash

import "golang.org/x/crypto/bcrypt"

type Hasher struct{ cost int }

func New(cost int) *Hasher {
	if cost < bcrypt.MinCost {
		cost = bcrypt.DefaultCost
	}
	return &Hasher{cost: cost}
}

func (h *Hasher) Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	return string(b), err
}

func (h *Hasher) Matches(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
