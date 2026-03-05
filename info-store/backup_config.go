// Meta DB Backup Configuration for CB-Spider
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2026.03.

package infostore

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// BackupConfig holds configuration for the meta DB backup scheduler.
type BackupConfig struct {
	Enabled   bool          // Enable/disable backup (default: true)
	Interval  time.Duration // Backup interval (default: 6h)
	BackupDir string        // Backup directory path (default: $CBSPIDER_ROOT/meta_db/backups)
	MaxCount  int           // Maximum number of backup files to retain (default: 10)
}

// Default values for backup configuration
const (
	DEFAULT_BACKUP_ENABLED   = true
	DEFAULT_BACKUP_INTERVAL  = 6 * time.Hour
	DEFAULT_BACKUP_MAX_COUNT = 10
	DEFAULT_BACKUP_DIR_NAME  = "backups"
)

// LoadBackupConfig loads backup configuration from environment variables.
// If any environment variable is not set or has an invalid value,
// the corresponding default value is used without producing an error.
func LoadBackupConfig() BackupConfig {
	cfg := BackupConfig{
		Enabled:   DEFAULT_BACKUP_ENABLED,
		Interval:  DEFAULT_BACKUP_INTERVAL,
		BackupDir: os.Getenv("CBSPIDER_ROOT") + "/meta_db/" + DEFAULT_BACKUP_DIR_NAME,
		MaxCount:  DEFAULT_BACKUP_MAX_COUNT,
	}

	// SPIDER_BACKUP_ENABLED
	if v := os.Getenv("SPIDER_BACKUP_ENABLED"); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "false", "off", "0", "no":
			cfg.Enabled = false
		case "true", "on", "1", "yes":
			cfg.Enabled = true
		default:
			cblog.Warnf("[MSB] Invalid SPIDER_BACKUP_ENABLED value '%s', using default: %v", v, DEFAULT_BACKUP_ENABLED)
		}
	}

	// SPIDER_BACKUP_INTERVAL
	if v := os.Getenv("SPIDER_BACKUP_INTERVAL"); v != "" {
		d, err := time.ParseDuration(strings.TrimSpace(v))
		if err != nil {
			cblog.Warnf("[MSB] Invalid SPIDER_BACKUP_INTERVAL value '%s', using default: %v", v, DEFAULT_BACKUP_INTERVAL)
		} else if d < 1*time.Minute {
			cblog.Warnf("[MSB] SPIDER_BACKUP_INTERVAL too small '%s', using minimum: 1m", v)
			cfg.Interval = 1 * time.Minute
		} else {
			cfg.Interval = d
		}
	}

	// SPIDER_BACKUP_DIR
	if v := os.Getenv("SPIDER_BACKUP_DIR"); v != "" {
		cfg.BackupDir = strings.TrimSpace(v)
	}

	// SPIDER_BACKUP_MAX_COUNT
	if v := os.Getenv("SPIDER_BACKUP_MAX_COUNT"); v != "" {
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil || n < 1 {
			cblog.Warnf("[MSB] Invalid SPIDER_BACKUP_MAX_COUNT value '%s', using default: %d", v, DEFAULT_BACKUP_MAX_COUNT)
		} else {
			cfg.MaxCount = n
		}
	}

	return cfg
}
