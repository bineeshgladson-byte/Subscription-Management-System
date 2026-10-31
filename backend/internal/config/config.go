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

	SMTPHost string
	SMTPPort int
	MailFrom string

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
		MailFrom:           get("MAIL_FROM", "billing@brightloop.local"),
		FrontendURL:        get("FRONTEND_URL", "http://localhost:5000"),
		CORSAllowedOrigins: split(get("CORS_ALLOWED_ORIGINS", "http://localhost:5000")),
		SeedAdminEmail:     os.Getenv("SEED_ADMIN_EMAIL"),
		SeedAdminPassword:  os.Getenv("SEED_ADMIN_PASSWORD"),
	}

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
