// Meta DB Backup Scheduler for CB-Spider
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2026.03.

package infostore

import (
	"context"
	"os"
	"time"
)

// StartBackupScheduler starts the background meta DB backup scheduler.
// It performs an immediate backup on startup, then runs periodically based on cfg.Interval.
// The scheduler stops gracefully when the provided context is cancelled.
// This function is non-blocking and runs in a goroutine.
func StartBackupScheduler(ctx context.Context, cfg BackupConfig) {
	if !cfg.Enabled {
		cblog.Info("[MSB] Meta DB backup is disabled.")
		return
	}

	cblog.Infof("[MSB] Meta DB Backup Scheduler started. interval=%v, maxCount=%d, dir=%s",
		cfg.Interval, cfg.MaxCount, cfg.BackupDir)

	go func() {
		// Perform an immediate backup on startup
		performBackup(cfg)

		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				performBackup(cfg)
			case <-ctx.Done():
				cblog.Info("[MSB] Meta DB Backup Scheduler stopped.")
				return
			}
		}
	}()
}

// performBackup executes a single backup cycle: backup + rotation.
func performBackup(cfg BackupConfig) {
	startTime := time.Now()

	// Check if source DB file exists
	if _, err := os.Stat(DB_FILE_PATH); os.IsNotExist(err) {
		cblog.Warnf("[MSB] Meta DB file not found: %s. Skipping backup.", DB_FILE_PATH)
		return
	}

	cblog.Info("[MSB] Starting meta DB backup...")

	backupPath, err := BackupMetaDB(cfg.BackupDir)
	if err != nil {
		cblog.Errorf("[MSB] Meta DB backup failed: %v", err)
		return
	}

	elapsed := time.Since(startTime)
	cblog.Infof("[MSB] Meta DB backup completed: %s (took %v)", backupPath, elapsed)

	// Rotate old backups
	if err := RotateBackups(cfg.BackupDir, cfg.MaxCount); err != nil {
		cblog.Errorf("[MSB] Backup rotation failed: %v", err)
	}
}
