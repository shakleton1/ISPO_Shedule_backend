package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ispo-schedule/internal/auth"
	"ispo-schedule/internal/config"
	"ispo-schedule/internal/db"
	"ispo-schedule/internal/httpapi"
	"ispo-schedule/internal/obs"
	"ispo-schedule/internal/pdf"
	"ispo-schedule/internal/schedule"
)

func Run() error {
	cfg, err := config.Load(config.LoadOptions{ConfigPath: "configs/config.yaml"})
	if err != nil {
		// Dev-friendly fallback
		cfg, err = config.Load(config.LoadOptions{ConfigPath: "configs/config.example.yaml"})
		if err != nil {
			return err
		}
	}

	obs.InitLogger(cfg.Log)

	gormDB, err := db.Open(cfg.DB)
	if err != nil {
		return err
	}

	scheduleRepo := schedule.NewRepository(gormDB)

	tokens, err := auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTL)
	if err != nil {
		return err
	}

	if err := bootstrapAdminIfNeeded(cfg, scheduleRepo); err != nil {
		return err
	}
	scheduleSvc := schedule.NewService(schedule.ServiceDeps{
		Repo:              scheduleRepo,
		SemesterStartDate: cfg.Schedule.SemesterStartDate,
		Now:               time.Now,
	})

	pdfEngine := pdf.NewEngine(pdf.EngineDeps{
		ChromeExecutablePath: cfg.PDF.ChromeExecutablePath,
		Timeout:              cfg.PDF.Timeout,
	})

	router := httpapi.NewRouter(httpapi.RouterDeps{
		Config:      cfg,
		ScheduleSvc: scheduleSvc,
		Repo:        scheduleRepo,
		PDF:         pdfEngine,
		Tokens:      tokens,
	})

	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-shutdownCtx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		return nil
	case err := <-errCh:
		if err == nil || err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("http server: %w", err)
	}
}

func bootstrapAdminIfNeeded(cfg *config.Config, repo *schedule.Repository) error {
	login := cfg.Auth.BootstrapAdminLogin
	pass := cfg.Auth.BootstrapAdminPassword
	if login == "" || pass == "" {
		return nil
	}
	_, err := repo.GetUserByLogin(login)
	if err == nil {
		return nil
	}

	hash, err := auth.HashPassword(pass)
	if err != nil {
		return err
	}

	u := auth.User{
		Login:        login,
		PasswordHash: hash,
		Role:         auth.RoleAdmin,
		GroupID:      nil,
		Subgroup:     nil,
	}
	// Create user; if record exists concurrently, ignore error.
	if err := repo.CreateUser(&u); err != nil {
		return err
	}
	return nil
}
