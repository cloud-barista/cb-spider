// Spider Status Board — public read-only view of CB-Spider test results.
// Receives run results pushed by SpiderWatch and serves them on the web.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloud-barista/cb-spider/spiderwatch/internal/statusboard"
	"github.com/cloud-barista/cb-spider/spiderwatch/internal/store"
	"github.com/cloud-barista/cb-spider/spiderwatch/internal/web"
)

var version = "dev"

func main() {
	cfgPath := flag.String("config", "conf/statusboard.yaml", "path to configuration file")
	flag.Parse()

	cfg, err := statusboard.Load(*cfgPath, func(newCfg *statusboard.Config) {
		// hot-reload: nothing to reconfigure at runtime for now
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := statusboard.SetupLogger(cfg)
	statusboard.SetLogger(log)
	log.Infof("Spider Status Board %s starting — port=%d", version, cfg.Server.Port)

	// Data store — results are stored under data/results/ relative to CWD.
	st, err := store.New("data/results")
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}

	// Template renderer — reuses SpiderWatch templates.
	renderer, err := web.NewRenderer("web/templates/*.html")
	if err != nil {
		log.Fatalf("failed to load templates: %v", err)
	}

	// HTTP server
	srv := statusboard.New(st, renderer)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)
	go func() {
		log.Infof("Spider Status Board listening on http://%s", addr)
		if err := srv.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Info("shutting down Spider Status Board...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Echo().Shutdown(ctx); err != nil {
		log.WithError(err).Error("server forced to shut down")
	}
	log.Info("Spider Status Board stopped")
}
