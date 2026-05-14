// Package runner provides the scheduler that triggers periodic test runs.
package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cloud-barista/cb-spider/spiderwatch/internal/config"
	"github.com/cloud-barista/cb-spider/spiderwatch/internal/model"
	"github.com/cloud-barista/cb-spider/spiderwatch/internal/store"
	"github.com/robfig/cron/v3"
)

// Scheduler wraps a cron scheduler and a Runner.
type Scheduler struct {
	cr          *cron.Cron
	runner      *Runner
	store       *store.Store
	currentCron string
	entryID     cron.EntryID
	onRunUpdate func(*model.RunResult)
	nextRunTime *time.Time
}

// NewScheduler creates a Scheduler.
func NewScheduler(r *Runner, s *store.Store, onRunUpdate func(*model.RunResult)) *Scheduler {
	return &Scheduler{
		cr:          cron.New(cron.WithSeconds()),
		runner:      r,
		store:       s,
		onRunUpdate: onRunUpdate,
	}
}

// Start starts the cron scheduler with the expression from config.
// If cfg.Scheduler.RunOnStartup is true, an immediate run is triggered.
func (s *Scheduler) Start(cfg *config.Config) {
	s.applyCron(cfg.Scheduler.Cron, cfg)
	s.cr.Start()
	log.Infof("scheduler: started with cron=%q", cfg.Scheduler.Cron)

	if cfg.Scheduler.RunOnStartup {
		go s.runOnce(cfg)
	}
}

// Reconfigure updates the cron expression (called on config hot-reload).
func (s *Scheduler) Reconfigure(cfg *config.Config) {
	if cfg.Scheduler.Cron == s.currentCron {
		return
	}
	log.Infof("scheduler: reconfiguring cron from %q to %q", s.currentCron, cfg.Scheduler.Cron)
	if s.entryID != 0 {
		s.cr.Remove(s.entryID)
	}
	s.applyCron(cfg.Scheduler.Cron, cfg)
}

// NextRunTime returns the next scheduled run time.
func (s *Scheduler) NextRunTime() *time.Time {
	if s.entryID == 0 {
		return nil
	}
	e := s.cr.Entry(s.entryID)
	t := e.Next
	if t.IsZero() {
		return nil
	}
	return &t
}

// TriggerNow manually triggers a run outside the cron schedule.
func (s *Scheduler) TriggerNow(cfg *config.Config) error {
	if s.runner.IsRunning() {
		return fmt.Errorf("scheduler: run already in progress")
	}
	go s.runOnce(cfg)
	return nil
}

// TriggerCleanupOnly triggers a cleanup-only run (skips all resource tests,
// only executes the cleanup step for each CSP).
func (s *Scheduler) TriggerCleanupOnly(cfg *config.Config) error {
	if s.runner.IsRunning() {
		return fmt.Errorf("scheduler: run already in progress")
	}
	cfgCopy := *cfg
	cfgCopy.Cleanup = "only"
	go s.runOnce(&cfgCopy)
	return nil
}

func (s *Scheduler) applyCron(expr string, _ *config.Config) {
	id, err := s.cr.AddFunc(expr, func() {
		s.runOnce(config.Get())
	})
	if err != nil {
		log.WithError(err).Errorf("scheduler: invalid cron expression %q", expr)
		return
	}
	s.entryID = id
	s.currentCron = expr
}

func (s *Scheduler) runOnce(cfg *config.Config) {
	log.Info("scheduler: triggering run")
	result, err := s.runner.Run(cfg, func(r *model.RunResult) {
		// Persist intermediate state
		if saveErr := s.store.Save(r); saveErr != nil {
			log.WithError(saveErr).Warn("scheduler: failed to save intermediate result")
		}
		if s.onRunUpdate != nil {
			s.onRunUpdate(r)
		}
	})
	if err != nil {
		log.WithError(err).Error("scheduler: run failed")
	}
	if result != nil {
		if saveErr := s.store.Save(result); saveErr != nil {
			log.WithError(saveErr).Error("scheduler: failed to save final result")
		}
		if s.onRunUpdate != nil {
			s.onRunUpdate(result)
		}
		// Push final result to Status Board (async, best-effort)
		if cfg.StatusBoard.URL != "" && cfg.StatusBoard.Token != "" {
			go pushToStatusBoard(cfg.StatusBoard.URL, cfg.StatusBoard.Token, result)
		}
	}
}

// pushToStatusBoard sends the completed run result to the public Status Board.
func pushToStatusBoard(boardURL, token string, result *model.RunResult) {
	data, err := json.Marshal(result)
	if err != nil {
		log.WithError(err).Warn("statusboard push: marshal failed")
		return
	}
	req, err := http.NewRequest(http.MethodPost, boardURL+"/api/v1/push", bytes.NewReader(data))
	if err != nil {
		log.WithError(err).Warn("statusboard push: create request failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Warnf("statusboard push: send failed (url=%s)", boardURL)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		log.Warnf("statusboard push: unexpected status %d from %s", resp.StatusCode, boardURL)
		return
	}
	log.Infof("statusboard push: run %s sent to %s", result.ID, boardURL)
}
