package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/caasmo/restinpieces"
	sqlitebackup "github.com/caasmo/restinpieces-sqlite-backup"
	"github.com/pelletier/go-toml/v2"
)

const JobTypeDbBackup = "db_backup"

func main() {
	// Create a simple slog text logger that outputs to stdout
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	dbPath := flag.String("dbpath", "", "Path to the SQLite DB")
	ageKeyPath := flag.String("age-key", "", "Path to the age identity (private key) file (required)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -db <db-path> -age-key <id-path>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Start the restinpieces application server with SQLite backup support.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *dbPath == "" || *ageKeyPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	// --- Create Database Pool ---
	dbPool, err := restinpieces.NewZombiezenPool(*dbPath)
	if err != nil {
		logger.Error("failed to create database pool", "path", *dbPath, "error", err)
		os.Exit(1)
	}
	defer func() {
		logger.Info("Closing database pool...")
		if err := dbPool.Close(); err != nil {
			logger.Error("Error closing database pool", "error", err)
		}
	}()

	// --- Initialize restinpieces ---
	app, srv, err := restinpieces.New(
		restinpieces.WithZombiezenPool(dbPool),
		restinpieces.WithAgeKeyPath(*ageKeyPath),
		restinpieces.WithLogger(logger),
	)
	if err != nil {
		logger.Error("failed to initialize restinpieces application", "error", err)
		os.Exit(1)
	}
	logger = app.Logger()

	// --- Load DB Backup Config from SecureConfigStore ---
	logger.Info("Loading DB backup configuration from database", "scope", sqlitebackup.ScopeDbBackup)
	encryptedTomlData, format, err := app.ConfigStore().Get(sqlitebackup.ScopeDbBackup, 0)
	if err != nil {
		logger.Error("failed to load DB backup config from DB", "scope", sqlitebackup.ScopeDbBackup, "error", err)
		os.Exit(1)
	}
	if len(encryptedTomlData) == 0 {
		logger.Error("DB backup config data loaded from DB is empty", "scope", sqlitebackup.ScopeDbBackup)
		os.Exit(1)
	}

	// Check if the format is TOML before unmarshalling
	if format != "toml" {
		logger.Error("DB backup config data is not in TOML format", "scope", sqlitebackup.ScopeDbBackup, "expected_format", "toml", "actual_format", format)
		os.Exit(1)
	}

	var backupCfg sqlitebackup.Config // Declare variable to hold the config
	if err := toml.Unmarshal(encryptedTomlData, &backupCfg); err != nil {
		logger.Error("failed to unmarshal DB backup TOML config", "scope", sqlitebackup.ScopeDbBackup, "error", err)
		os.Exit(1)
	}
	logger.Info("Successfully unmarshalled DB backup config", "scope", sqlitebackup.ScopeDbBackup)

	// --- Create and Register Backup Handler ---
	dbBackupHandler := sqlitebackup.NewHandler(&backupCfg)
	err = srv.AddJobHandler(JobTypeDbBackup, dbBackupHandler)
	if err != nil {
		logger.Error("Failed to register database backup job handler", "job_type", JobTypeDbBackup, "error", err)
		os.Exit(1)
	}
	logger.Info("Registered database backup job handler", "job_type", JobTypeDbBackup)

	srv.Run()

	logger.Info("Server shut down gracefully.")
}
