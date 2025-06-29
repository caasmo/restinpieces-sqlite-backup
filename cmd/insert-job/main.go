
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/caasmo/restinpieces"
	"github.com/caasmo/restinpieces/db"
	"github.com/caasmo/restinpieces/db/zombiezen"
)

const JobTypeDbBackup = "db_backup"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	dbPath := flag.String("dbpath", "", "Path to the SQLite DB file (required)")
	interval := flag.Duration("interval", 24*time.Hour, "Interval for the recurrent backup job (e.g., '24h', '1h30m')")
	flag.Parse()

	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "Error: -dbpath is required")
		flag.Usage()
		os.Exit(1)
	}

	logger.Info("Connecting to database...", "path", *dbPath)

	// Use the framework's method to create a database pool
	pool, err := restinpieces.NewZombiezenPool(*dbPath)
	if err != nil {
		logger.Error("Failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Use the framework's db implementation
	dbConn, err := zombiezen.New(pool)
	if err != nil {
		logger.Error("Failed to create db connection", "error", err)
		os.Exit(1)
	}

	// Define the recurrent job
	// No payload is needed for this job type
	payload, err := json.Marshal(struct{}{})
	if err != nil {
		logger.Error("Failed to marshal empty payload", "error", err)
		os.Exit(1)
	}

	newJob := db.Job{
		Type:      JobTypeDbBackup,
		Payload:   payload,
		RunAt:     time.Now(), // It will run as soon as a worker picks it up
		Recurrent: true,
		Interval:  interval.String(),
	}

	logger.Info("Inserting recurrent backup job into database", "type", newJob.Type, "interval", newJob.Interval)

	// Insert the job using the DbQueue interface
	if err := dbConn.InsertJob(newJob); err != nil {
		logger.Error("Failed to insert job", "error", err)
		os.Exit(1)
	}

	logger.Info("Successfully inserted backup job. The server will pick it up on its next cycle.")
}
