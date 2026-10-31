// Command api starts the Subly HTTP server and the billing scheduler.
//
// Wiring order: config → database → repositories → services → handlers → router.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brightloop/subscriptions/backend/internal/config"
	"github.com/brightloop/subscriptions/backend/internal/database"
	"github.com/brightloop/subscriptions/backend/internal/handler"
	"github.com/brightloop/subscriptions/backend/internal/jobs"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/internal/router"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/clock"
	"github.com/brightloop/subscriptions/backend/pkg/hash"
	"github.com/brightloop/subscriptions/backend/pkg/jwtutil"
	"github.com/brightloop/subscriptions/backend/pkg/mailer"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := database.Connect(cfg.DatabaseURL, false)
	if err != nil {
		return err
	}

	clk := clock.Real{Loc: cfg.Location}
	jwtMgr := jwtutil.New(cfg.JWTSecret, cfg.JWTTTL)
	hasher := hash.New(cfg.BcryptCost)
	mail := mailer.NewSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.MailFrom, cfg.MailFromName, cfg.SMTPUsername, cfg.SMTPPassword)

	// Repositories
	userRepo := repository.NewUserRepo()
	customerRepo := repository.NewCustomerRepo()
	resetTokenRepo := repository.NewResetTokenRepo()

	// Services
	authSvc := service.NewAuthService(service.AuthDeps{
		DB: db, Users: userRepo, Customers: customerRepo, ResetTokens: resetTokenRepo,
		Hasher: hasher, JWT: jwtMgr, Clock: clk, Mailer: mail,
		FrontendURL: cfg.FrontendURL, ResetTTL: cfg.ResetTokenTTL, ShowCodesInEmail: cfg.EmailShowCodes,
	})
	staffSvc := service.NewStaffService(db, userRepo, hasher, authSvc, cfg.InviteTokenTTL)
	customerSvc := service.NewCustomerService(db, customerRepo, userRepo, clk)
	taxSvc := service.NewTaxService(db, repository.NewTaxRepo())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if created, err := authSvc.EnsureAdmin(ctx, cfg.SeedAdminEmail, cfg.SeedAdminPassword); err != nil {
		return err
	} else if created {
		slog.Info("created first admin account", "email", cfg.SeedAdminEmail)
	}

	// HTTP
	engine := router.New(router.Deps{
		Config:   cfg,
		DB:       db,
		JWT:      jwtMgr,
		Sessions: authSvc,
		Auth:     handler.NewAuthHandler(authSvc),
		Staff:    handler.NewStaffHandler(staffSvc),

		Customers: handler.NewCustomerHandler(customerSvc),
		Taxes:     handler.NewTaxHandler(taxSvc),
	})
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: engine, ReadHeaderTimeout: 10 * time.Second}

	// Background jobs
	sched := jobs.NewScheduler(clk)
	if err := sched.Start(cfg.BillingCron); err != nil {
		return err
	}

	go func() {
		slog.Info("api listening", "port", cfg.Port, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	sched.Stop(shutdownCtx)
	return srv.Shutdown(shutdownCtx)
}
