// Meta DB Backup Logic for CB-Spider
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2026.03.

package infostore

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const BACKUP_FILE_PREFIX = "cb-spider_backup_"
const BACKUP_FILE_SUFFIX = ".db"
const BACKUP_TIME_FORMAT = "20060102_150405"

// BackupMetaDB performs an online backup of the meta DB using SQLite3 VACUUM INTO.
// This is safe to call while the DB is being read/written by the Spider server.
// Returns the backup file path on success.
func BackupMetaDB(backupDir string) (string, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory '%s': %w", backupDir, err)
	}

	// Generate backup file name with timestamp
	timestamp := time.Now().Format(BACKUP_TIME_FORMAT)
	backupFileName := BACKUP_FILE_PREFIX + timestamp + BACKUP_FILE_SUFFIX
	backupFilePath := filepath.Join(backupDir, backupFileName)

	// Open a direct connection to the source DB for VACUUM INTO
	srcDB, err := sql.Open("sqlite3", DB_FILE_PATH+"?_busy_timeout=60000")
	if err != nil {
		return "", fmt.Errorf("failed to open source DB for backup: %w", err)
	}
	defer srcDB.Close()

	// Execute VACUUM INTO to create a consistent backup
	// VACUUM INTO creates a new optimized copy of the database without affecting ongoing operations
	_, err = srcDB.Exec("VACUUM INTO ?", backupFilePath)
	if err != nil {
		// Clean up partial backup file if it exists
		os.Remove(backupFilePath)
		return "", fmt.Errorf("VACUUM INTO failed: %w", err)
	}

	return backupFilePath, nil
}

// RotateBackups enforces the maximum backup count by deleting the oldest backups.
// It keeps at most maxCount backup files in the backupDir.
func RotateBackups(backupDir string, maxCount int) error {
	backups, err := listBackupFiles(backupDir)
	if err != nil {
		return fmt.Errorf("failed to list backup files: %w", err)
	}

	// If within limit, nothing to do
	if len(backups) <= maxCount {
		return nil
	}

	// Sort by name ascending (oldest first, since names contain timestamps)
	sort.Strings(backups)

	// Delete oldest files exceeding maxCount
	deleteCount := len(backups) - maxCount
	for i := 0; i < deleteCount; i++ {
		filePath := filepath.Join(backupDir, backups[i])
		if err := os.Remove(filePath); err != nil {
			cblog.Warnf("[MSB] Failed to delete old backup '%s': %v", backups[i], err)
		} else {
			cblog.Infof("[MSB] Deleted old backup: %s", backups[i])
		}
	}

	return nil
}

// listBackupFiles returns a sorted list of backup file names in the given directory.
func listBackupFiles(backupDir string) ([]string, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var backups []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, BACKUP_FILE_PREFIX) && strings.HasSuffix(name, BACKUP_FILE_SUFFIX) {
			backups = append(backups, name)
		}
	}

	return backups, nil
}
