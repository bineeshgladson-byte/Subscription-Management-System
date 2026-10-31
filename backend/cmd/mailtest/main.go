// Command mailtest sends ONE test email using the SMTP settings in .env, and
// prints exactly what went wrong if it fails. Use it to check Gmail setup:
//
//	make test-email TO=you@gmail.com
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/brightloop/subscriptions/backend/internal/config"
	"github.com/brightloop/subscriptions/backend/pkg/brand"
	"github.com/brightloop/subscriptions/backend/pkg/mailer"
)

func main() {
	if len(os.Args) < 2 || !strings.Contains(os.Args[1], "@") {
		fmt.Println("usage: mailtest you@example.com")
		os.Exit(2)
	}
	to := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		fail("config", err)
	}

	auth := "none (fine for Mailpit)"
	if cfg.SMTPUsername != "" {
		auth = cfg.SMTPUsername + " / password " + mask(cfg.SMTPPassword)
	}
	fmt.Printf("SMTP server : %s:%d\nLogin       : %s\nFrom        : %s <%s>\nTo          : %s\n\n",
		cfg.SMTPHost, cfg.SMTPPort, auth, cfg.MailFromName, cfg.MailFrom, to)

	m := mailer.NewSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.MailFrom, cfg.MailFromName, cfg.SMTPUsername, cfg.SMTPPassword)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = m.Send(ctx, mailer.Message{
		To:      to,
		Subject: brand.Name + " test email",
		Body: "This is a test email from " + brand.Name + ".\n\n" +
			"If you can read this, password reset and staff invite emails will be delivered too.\n" +
			"Sent at " + time.Now().In(cfg.Location).Format("2 Jan 2006 15:04:05 MST") + "\n" + brand.EmailSignature,
	})
	if err != nil {
		fail("send", err)
	}
	fmt.Println("✅ Sent. Check the inbox of", to, "(and the Spam folder).")
	if cfg.SMTPHost == "mailpit" || cfg.SMTPHost == "localhost" {
		fmt.Println("   You're using Mailpit, so it's at http://localhost:8025, not a real inbox.")
	}
}

func fail(step string, err error) {
	fmt.Printf("❌ %s failed: %v\n\n", step, err)
	msg := err.Error()
	switch {
	case strings.Contains(msg, "535"), strings.Contains(msg, "Username and Password not accepted"):
		fmt.Println("Gmail rejected the login. Use an App Password (myaccount.google.com/apppasswords),")
		fmt.Println("not your normal Gmail password, paste it without spaces, and check SMTP_USERNAME.")
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "i/o timeout"), strings.Contains(msg, "connection refused"):
		fmt.Println("Couldn't reach the mail server. Check SMTP_HOST/SMTP_PORT (Gmail: smtp.gmail.com / 587).")
		fmt.Println("Some Wi-Fi networks block port 587; try a phone hotspot.")
	case strings.Contains(msg, "unencrypted connection"):
		fmt.Println("The server doesn't offer TLS on this port. For Gmail use port 587.")
	case strings.Contains(msg, "no such host"):
		fmt.Println("SMTP_HOST is wrong or there's no internet connection.")
	}
	os.Exit(1)
}

func mask(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}
