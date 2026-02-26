package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ispo-schedule/internal/config"
	"ispo-schedule/internal/db"
	"ispo-schedule/internal/httpapi"
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

	gormDB, err := db.Open(cfg.DB)
	if err != nil {
		return err
	}

	scheduleRepo := schedule.NewRepository(gormDB)
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
