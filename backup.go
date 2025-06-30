
package sqlitebackup

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/caasmo/restinpieces/db"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

const (
	ScopeDbBackup  = "sqlite_backup"
	StrategyVacuum = "vacuum"
	StrategyOnline = "online"
)

// Config defines the settings for the backup job.
type Config struct {
	SourcePath string `toml:"source_path"`
	BackupDir  string `toml:"backup_dir"`
	Strategy   string `toml:"strategy"`
	PagesPerStep  int `toml:"pages_per_step"`
	SleepInterval time.Duration `toml:"sleep_interval"`
}

// Handler handles database backup jobs
type Handler struct {
	cfg    *Config
	logger *slog.Logger
}

// NewHandler creates a new Handler
func NewHandler(cfg *Config, logger *slog.Logger) *Handler {
	if cfg == nil || logger == nil {
		panic("NewHandler: received nil config or logger")
	}
	return &Handler{
		cfg:    cfg,
		logger: logger.With("job_handler", "sqlite_backup"),
	}
}

// GenerateBlueprintConfig creates a default configuration for a new setup.
func GenerateBlueprintConfig() Config {
	return Config{
		SourcePath:    "/path/to/your/database.db",
		BackupDir:     "/path/to/your/backups",
		Strategy:      StrategyVacuum,
		PagesPerStep:  100,
		SleepInterval: 10 * time.Millisecond,
	}
}

// Handle implements the JobHandler interface for database backups
func (h *Handler) Handle(ctx context.Context, job db.Job) error {
	// Define paths
	sourceDbPath := h.cfg.SourcePath
	backupDir := h.cfg.BackupDir
	tempBackupPath := filepath.Join(os.TempDir(), fmt.Sprintf("backup-%d.db", time.Now().UnixNano()))
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	finalBackupName := fmt.Sprintf("db-backup-%s.db.gz", timestamp)
	finalBackupPath := filepath.Join(backupDir, finalBackupName)
	manifestPath := filepath.Join(backupDir, "latest.txt")

	h.logger.Info("Starting database backup process", "source", sourceDbPath, "strategy", h.cfg.Strategy)

	// Dispatch to the chosen backup strategy
	var backupErr error
	switch h.cfg.Strategy {
	case StrategyOnline:
		backupErr = h.onlineBackup(sourceDbPath, tempBackupPath)
	case StrategyVacuum, "": // Default to "vacuum"
		backupErr = h.vacuumInto(sourceDbPath, tempBackupPath)
	default:
		return fmt.Errorf("unknown backup strategy: %q", h.cfg.Strategy)
	}

	if backupErr != nil {
		return fmt.Errorf("backup creation failed: %w", backupErr)
	}
	defer os.Remove(tempBackupPath)
	h.logger.Info("Successfully created temporary backup database", "path", tempBackupPath)

	// Gzip the temporary file
	if err := h.compressFile(tempBackupPath, finalBackupPath); err != nil {
		return fmt.Errorf("failed to gzip backup file: %w", err)
	}
	h.logger.Info("Successfully compressed backup", "path", finalBackupPath)

	// Update the manifest file
	if err := os.WriteFile(manifestPath, []byte(finalBackupName), 0644); err != nil {
		return fmt.Errorf("failed to update manifest file: %w", err)
	}
	h.logger.Info("Successfully updated manifest file", "manifest", manifestPath)

	h.logger.Info("Database backup process completed successfully")
	return nil
}

// vacuumInto creates a clean, defragmented copy of the database.
func (h *Handler) vacuumInto(sourcePath, destPath string) error {
	h.logger.Info("Starting 'vacuum' backup. Writers will be blocked during this operation.")
	sourceConn, err := sqlite.OpenConn(sourcePath, sqlite.OpenReadOnly)
	if err != nil {
		return fmt.Errorf("failed to open source db for vacuum: %w", err)
	}
	defer sourceConn.Close()

	stmt, err := sourceConn.Prepare(fmt.Sprintf("VACUUM INTO '%s';", destPath))
	if err != nil {
		return fmt.Errorf("failed to prepare vacuum statement: %w", err)
	}
	defer stmt.Finalize()

	if _, err := stmt.Step(); err != nil {
		return fmt.Errorf("failed to execute vacuum statement: %w", err)
	}
	return nil
}

// onlineBackup performs a live backup using the SQLite Online Backup API.
func (h *Handler) onlineBackup(sourcePath, destPath string) error {
	h.logger.Info("Starting 'online' backup. This may take longer but will not block writers.")
	pagesPerStep := h.cfg.PagesPerStep
	if pagesPerStep <= 0 {
		pagesPerStep = 100
	}
	sleepInterval := h.cfg.SleepInterval
	if sleepInterval < 0 {
		sleepInterval = 10 * time.Millisecond
	}

	srcConn, err := sqlitex.Open(sourcePath, sqlite.OpenReadOnly, "")
	if err != nil {
		return fmt.Errorf("failed to open source db for online backup: %w", err)
	}
	defer srcConn.Close()

	destConn, err := sqlitex.Open(destPath, sqlite.OpenCreate|sqlite.OpenReadWrite, "")
	if err != nil {
		return fmt.Errorf("failed to create destination db for online backup: %w", err)
	}
	defer destConn.Close()

	backup, err := sqlitex.NewBackup(destConn, "main", srcConn, "main")
	if err != nil {
		return fmt.Errorf("failed to initialize backup: %w", err)
	}
	defer backup.Finish()

	h.logger.Info("Starting online backup copy", "pages_per_step", pagesPerStep, "sleep_interval", sleepInterval)
	for {
		done, err := backup.Step(pagesPerStep)
		if err != nil {
			return fmt.Errorf("backup step failed: %w", err)
		}
		if done {
			h.logger.Info("Online backup copy completed successfully.")
			return nil
		}
		if sleepInterval > 0 {
			time.Sleep(sleepInterval)
		}
	}
}

// compressFile reads a source file, compresses it with gzip, and writes to a destination file.
func (h *Handler) compressFile(sourcePath, destPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file for compression: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file for compression: %w", err)
	}
	defer destFile.Close()

	gzipWriter := gzip.NewWriter(destFile)
	defer gzipWriter.Close()

	if _, err := io.Copy(gzipWriter, sourceFile); err != nil {
		return fmt.Errorf("failed to copy and compress data: %w", err)
	}

	return nil
}
