// Package config reads all settings from environment variables.
// Nothing secret is ever hard-coded; see .env.example at the repo root.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv      string
	Port        string
	DatabaseURL string

	JWTSecret  string
	JWTTTL     time.Duration
	BcryptCost int

	Location *time.Location // business timezone used for every "today"

	BillingCron        string
	InvoiceAutoConfirm bool

	SMTPHost     string
	SMTPPort     int
	SMTPUsername string // empty for Mailpit
	SMTPPassword string
	MailFrom     string
	MailFromName string
	// EmailShowCodes puts reset/invite codes in email text for Postman testing.
	EmailShowCodes bool

	ResetTokenTTL  time.Duration // forgot-password links
	InviteTokenTTL time.Duration // "set your password" links for new staff

	FrontendURL        string
	CORSAllowedOrigins []string

	SeedAdminEmail    string
	SeedAdminPassword string
}

func (c Config) IsDev() bool { return c.AppEnv == "development" }

// Load reads .env (if present, for local runs) and then the process environment.
func Load() (Config, error) {
	_ = godotenv.Load(".env", "../.env") // optional; docker-compose passes env directly

	loc, err := time.LoadLocation(get("BUSINESS_TIMEZONE", "Asia/Kolkata"))
	if err != nil {
		return Config{}, fmt.Errorf("BUSINESS_TIMEZONE: %w", err)
	}

	cfg := Config{
		AppEnv:             get("APP_ENV", "development"),
		Port:               get("API_PORT", "8080"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		JWTTTL:             time.Duration(getInt("JWT_TTL_HOURS", 8)) * time.Hour,
		BcryptCost:         getInt("BCRYPT_COST", 12),
		Location:           loc,
		BillingCron:        get("BILLING_CRON", "@every 1h"),
		InvoiceAutoConfirm: get("INVOICE_AUTO_CONFIRM", "true") == "true",
		SMTPHost:           get("SMTP_HOST", "localhost"),
		SMTPPort:           getInt("SMTP_PORT", 1025),
		SMTPUsername:       os.Getenv("SMTP_USERNAME"),
		SMTPPassword:       os.Getenv("SMTP_PASSWORD"),
		MailFrom:           get("MAIL_FROM", "billing@brightloop.local"),
		MailFromName:       get("MAIL_FROM_NAME", "Subly"),
		ResetTokenTTL:      time.Duration(getInt("RESET_TOKEN_TTL_MINUTES", 30)) * time.Minute,
		InviteTokenTTL:     time.Duration(getInt("INVITE_TOKEN_TTL_HOURS", 48)) * time.Hour,
		FrontendURL:        get("FRONTEND_URL", "http://localhost:5000"),
		CORSAllowedOrigins: split(get("CORS_ALLOWED_ORIGINS", "http://localhost:5000")),
		SeedAdminEmail:     os.Getenv("SEED_ADMIN_EMAIL"),
		SeedAdminPassword:  os.Getenv("SEED_ADMIN_PASSWORD"),
	}

	// Default: show codes in development only. Override with EMAIL_SHOW_CODES=true|false.
	cfg.EmailShowCodes = get("EMAIL_SHOW_CODES", fmt.Sprint(cfg.IsDev())) == "true"

	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return cfg, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	return cfg, nil
}

func get(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return v
	}
	return def
}

func split(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
