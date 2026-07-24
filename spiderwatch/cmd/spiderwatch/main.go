// SpiderWatch - CB-Spider Status Board
// Periodically tests all CSP resources via CB-Spider and serves results on the web.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cloud-barista/cb-spider/spiderwatch/internal/config"
	"github.com/cloud-barista/cb-spider/spiderwatch/internal/runner"
	"github.com/cloud-barista/cb-spider/spiderwatch/internal/store"
	"github.com/cloud-barista/cb-spider/spiderwatch/internal/web"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func main() {
	cfgPath := flag.String("config", "conf/spiderwatch.yaml", "path to configuration file")
	flag.Parse()

	// Scheduler and store are created after config load; the hot-reload callback
	// references sched, so we declare it before calling config.Load.
	var sched *runner.Scheduler

	cfg, err := config.Load(*cfgPath, func(newCfg *config.Config) {
		setupLogger(newCfg)
		if sched != nil {
			sched.Reconfigure(newCfg)
		}
		log.Info("configuration hot-reloaded")
	})
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	setupLogger(cfg)
	log.Infof("SpiderWatch starting - port=%d cron=%q", cfg.Server.Port, cfg.Scheduler.Cron)

	// Data store
	st, err := store.New("data/results")
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}

	// Runner + Scheduler
	r := runner.New()
	sched = runner.NewScheduler(r, st, nil)

	// Web server
	renderer, err := web.NewRenderer("web/templates/*.html")
	if err != nil {
		log.Fatalf("failed to load templates: %v", err)
	}
	srv := web.New(st, sched, r, renderer, *cfgPath)

	// Start scheduler
	sched.Start(cfg)

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)
	go func() {
		log.Infof("SpiderWatch listening on http://%s", addr)
		if err := srv.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Info("shutting down SpiderWatch...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Echo().Shutdown(ctx); err != nil {
		log.WithError(err).Error("server shutdown error")
	}
	log.Info("SpiderWatch stopped")
}

func setupLogger(cfg *config.Config) {
	lvl, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	log.SetLevel(lvl)
	logrus.SetLevel(lvl)
	if cfg.Log.File != "" {
		if mkErr := os.MkdirAll(filepath.Dir(cfg.Log.File), 0o755); mkErr == nil {
			f, fErr := os.OpenFile(cfg.Log.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if fErr == nil {
				log.SetOutput(f)
			}
		}
	}
}
