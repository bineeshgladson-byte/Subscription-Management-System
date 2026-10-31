// Package brand holds the product name shown to users (emails, API docs).
// Brightloop Software is the company; Subly is the app it uses.
package brand

const (
	Name    = "Subly"
	Tagline = "Subscriptions. Billing. Simplified."
	Company = "Brightloop Software"
)

// EmailSignature is appended to every email the app sends.
const EmailSignature = "\n--\n" + Name + " · " + Tagline + "\n" + Company + "\n"
