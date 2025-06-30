
package sqlitebackup

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	"log/slog"

	"github.com/caasmo/restinpieces/db"
	"zombiezen.com/go/sqlite"
)

const ScopeDbBackup = "sqlite_backup"

type Config struct {
	SourcePath string `toml:"source_path"`
	BackupDir  string `toml:"backup_dir"`
}

// Handler handles database backup jobs
type Handler struct {
	cfg *Config
}

// NewHandler creates a new Handler
func NewHandler(cfg *Config) *Handler {
	return &Handler{
		cfg: cfg,
	}
}

func GenerateBlueprintConfig() Config {
	return Config{
		SourcePath: "/path/to/your/database.db",
		BackupDir:  "/path/to/your/backups",
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

	slog.Info("Starting database backup process", "source", sourceDbPath, "destination", finalBackupPath)

	// 1. Perform VACUUM INTO a temporary file
	if err := h.vacuumInto(sourceDbPath, tempBackupPath); err != nil {
		return fmt.Errorf("failed to perform vacuum into: %w", err)
	}
	defer os.Remove(tempBackupPath) // Ensure temp file is cleaned up

	slog.Info("Successfully created temporary vacuumed database", "path", tempBackupPath)

	// 2. Gzip the temporary file to its final destination
	if err := h.compressFile(tempBackupPath, finalBackupPath); err != nil {
		return fmt.Errorf("failed to gzip backup file: %w", err)
	}

	slog.Info("Successfully compressed backup", "path", finalBackupPath)

	// 3. Update the manifest file
	if err := os.WriteFile(manifestPath, []byte(finalBackupName), 0644); err != nil {
		return fmt.Errorf("failed to update manifest file: %w", err)
	}

	slog.Info("Successfully updated manifest file", "manifest", manifestPath)

	// (Optional) 4. Clean up old backups
	// This part can be added later if needed.

	slog.Info("Database backup process completed successfully")
	return nil
}

// vacuumInto creates a clean copy of the database using VACUUM INTO.
func (h *Handler) vacuumInto(sourcePath, destPath string) error {
	// Open a read-only connection to the source database.
	sourceConn, err := sqlite.OpenConn(sourcePath, sqlite.OpenReadOnly)
	if err != nil {
		return fmt.Errorf("failed to open source db for vacuum: %w", err)
	}
	defer sourceConn.Close()

	// Execute the VACUUM INTO command. The destination path must be a single-quoted string literal.
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
