package backup

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Ameprizzo/go-watchdog/internal/database"
)

// BackupService handles database backups and recovery
type BackupService struct {
	repos      *database.Repositories
	backupDir  string
	maxBackups int
}

// BackupInfo contains metadata about a backup
type BackupInfo struct {
	Filename       string
	Path           string
	Size           int64
	CreatedAt      time.Time
	CompressedSize int64
}

// BackupResult contains details about a backup operation
type BackupResult struct {
	Timestamp      time.Time
	BackupFile     string
	BackupSize     int64
	CompressedSize int64
	Duration       time.Duration
	Success        bool
	Error          string
}

// NewBackupService creates a new backup service
func NewBackupService(repos *database.Repositories, backupDir string, maxBackups int) (*BackupService, error) {
	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	log.Printf("✅ Backup service initialized at: %s (max backups: %d)", backupDir, maxBackups)

	return &BackupService{
		repos:      repos,
		backupDir:  backupDir,
		maxBackups: maxBackups,
	}, nil
}

// CreateBackup creates a compressed backup of the database
func (b *BackupService) CreateBackup() (*BackupResult, error) {
	result := &BackupResult{
		Timestamp: time.Now(),
	}

	startTime := time.Now()

	// Source database file
	sourceDB := "watchdog.db"
	if _, err := os.Stat(sourceDB); err != nil {
		result.Error = fmt.Sprintf("database file not found: %v", err)
		return result, err
	}

	// Create backup filename with timestamp
	backupName := fmt.Sprintf("watchdog_backup_%s.db.gz", time.Now().Format("20060102_150405"))
	backupPath := filepath.Join(b.backupDir, backupName)

	// Open source file
	sourceFile, err := os.Open(sourceDB)
	if err != nil {
		result.Error = fmt.Sprintf("failed to open source database: %v", err)
		return result, err
	}
	defer sourceFile.Close()

	// Get source file size
	sourceInfo, _ := sourceFile.Stat()
	result.BackupSize = sourceInfo.Size()

	// Create destination backup file
	backupFile, err := os.Create(backupPath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create backup file: %v", err)
		return result, err
	}
	defer backupFile.Close()

	// Compress and write to backup file
	gzWriter := gzip.NewWriter(backupFile)
	defer gzWriter.Close()

	if _, err := io.Copy(gzWriter, sourceFile); err != nil {
		result.Error = fmt.Sprintf("failed to compress backup: %v", err)
		return result, err
	}

	// Get compressed size
	backupInfo, _ := backupFile.Stat()
	result.CompressedSize = backupInfo.Size()
	result.BackupFile = backupName
	result.Duration = time.Since(startTime)
	result.Success = true

	log.Printf("✅ Database backup created: %s (Original: %.2f MB, Compressed: %.2f MB)",
		backupName,
		float64(result.BackupSize)/1024/1024,
		float64(result.CompressedSize)/1024/1024)

	// Clean up old backups
	if err := b.rotateBackups(); err != nil {
		log.Printf("⚠️ Failed to rotate old backups: %v", err)
	}

	// Audit log
	b.logAudit("backup_created", "backup", 0,
		fmt.Sprintf(`{"filename": "%s", "size": %d, "compressed_size": %d}`,
			backupName, result.BackupSize, result.CompressedSize))

	return result, nil
}

// RestoreBackup restores a database from a backup file
func (b *BackupService) RestoreBackup(backupFile string) (*BackupResult, error) {
	result := &BackupResult{
		Timestamp: time.Now(),
	}

	startTime := time.Now()

	backupPath := filepath.Join(b.backupDir, backupFile)

	// Check if backup exists
	if _, err := os.Stat(backupPath); err != nil {
		result.Error = fmt.Sprintf("backup file not found: %v", err)
		return result, err
	}

	// Get backup info
	backupInfo, _ := os.Stat(backupPath)
	result.CompressedSize = backupInfo.Size()

	// Open backup file
	backupFileHandle, err := os.Open(backupPath)
	if err != nil {
		result.Error = fmt.Sprintf("failed to open backup file: %v", err)
		return result, err
	}
	defer backupFileHandle.Close()

	// Create a temporary restore file
	tempDB := "watchdog_restore_temp.db"
	tempFile, err := os.Create(tempDB)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create temp restore file: %v", err)
		return result, err
	}
	defer tempFile.Close()

	// Decompress backup
	gzReader, err := gzip.NewReader(backupFileHandle)
	if err != nil {
		result.Error = fmt.Sprintf("failed to decompress backup: %v", err)
		return result, err
	}
	defer gzReader.Close()

	// Write decompressed data to temp file
	if _, err := io.Copy(tempFile, gzReader); err != nil {
		result.Error = fmt.Sprintf("failed to decompress backup: %v", err)
		os.Remove(tempDB)
		return result, err
	}

	tempInfo, _ := tempFile.Stat()
	result.BackupSize = tempInfo.Size()

	// Close temp file before moving
	tempFile.Close()

	// Backup current database
	currentDB := "watchdog.db"
	backupCurrent := fmt.Sprintf("watchdog_current_%s.db", time.Now().Format("20060102_150405"))

	if _, err := os.Stat(currentDB); err == nil {
		// Current DB exists, create a backup of it first
		if err := os.Rename(currentDB, backupCurrent); err != nil {
			result.Error = fmt.Sprintf("failed to backup current database: %v", err)
			os.Remove(tempDB)
			return result, err
		}
	}

	// Restore by renaming temp file to current DB
	if err := os.Rename(tempDB, currentDB); err != nil {
		result.Error = fmt.Sprintf("failed to restore database: %v", err)
		// Try to restore current DB from backup
		if _, err := os.Stat(backupCurrent); err == nil {
			os.Rename(backupCurrent, currentDB)
		}
		return result, err
	}

	result.BackupFile = backupFile
	result.Duration = time.Since(startTime)
	result.Success = true

	log.Printf("✅ Database restored from backup: %s (Size: %.2f MB)",
		backupFile, float64(result.BackupSize)/1024/1024)

	// Audit log
	b.logAudit("backup_restored", "backup", 0,
		fmt.Sprintf(`{"filename": "%s", "size": %d}`, backupFile, result.BackupSize))

	return result, nil
}

// ListBackups returns a list of available backups
func (b *BackupService) ListBackups() ([]BackupInfo, error) {
	var backups []BackupInfo

	entries, err := os.ReadDir(b.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".gz" {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			backups = append(backups, BackupInfo{
				Filename:  entry.Name(),
				Path:      filepath.Join(b.backupDir, entry.Name()),
				Size:      info.Size(),
				CreatedAt: info.ModTime(),
			})
		}
	}

	// Sort by creation time, newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// GetBackupSize returns the total size of all backups
func (b *BackupService) GetBackupSize() (int64, error) {
	backups, err := b.ListBackups()
	if err != nil {
		return 0, err
	}

	var totalSize int64
	for _, backup := range backups {
		totalSize += backup.Size
	}

	return totalSize, nil
}

// rotateBackups removes old backups keeping only maxBackups
func (b *BackupService) rotateBackups() error {
	backups, err := b.ListBackups()
	if err != nil {
		return err
	}

	// If we have more than max backups, delete the oldest ones
	if len(backups) > b.maxBackups {
		for i := b.maxBackups; i < len(backups); i++ {
			if err := os.Remove(backups[i].Path); err != nil {
				log.Printf("⚠️ Failed to delete old backup %s: %v", backups[i].Filename, err)
			} else {
				log.Printf("🗑️ Deleted old backup: %s", backups[i].Filename)
			}
		}
	}

	return nil
}

// VerifyBackupIntegrity checks if a backup file is valid
func (b *BackupService) VerifyBackupIntegrity(backupFile string) (bool, error) {
	backupPath := filepath.Join(b.backupDir, backupFile)

	// Check file exists
	if _, err := os.Stat(backupPath); err != nil {
		return false, fmt.Errorf("backup file not found: %w", err)
	}

	// Try to open and read the gzip header
	file, err := os.Open(backupPath)
	if err != nil {
		return false, fmt.Errorf("failed to open backup: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return false, fmt.Errorf("invalid gzip file: %w", err)
	}
	defer gzReader.Close()

	// Try to read some data
	buf := make([]byte, 1024)
	_, err = gzReader.Read(buf)
	if err != nil && err.Error() != "EOF" {
		return false, fmt.Errorf("failed to read backup content: %w", err)
	}

	log.Printf("✅ Backup integrity verified: %s", backupFile)
	return true, nil
}

// logAudit records an audit log entry
func (b *BackupService) logAudit(action, entityType string, entityID int, details string) {
	auditLog := &database.AuditLog{
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		UserID:     "system",
		NewValue:   details,
		Timestamp:  time.Now(),
	}
	if err := b.repos.AuditLog.Create(auditLog); err != nil {
		log.Printf("❌ Failed to create audit log: %v", err)
	}
}
