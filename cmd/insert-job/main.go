
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
	interval := flag.String("interval", "", "Interval for the recurrent backup job (e.g., '24h', '1h30m') (required)")
	scheduledForStr := flag.String("scheduledFor", "", "Start time for the job in RFC3339 format (e.g., '2025-07-01T10:00:00Z') (required)")
	flag.Parse()

	if *dbPath == "" || *interval == "" || *scheduledForStr == "" {
		fmt.Fprintln(os.Stderr, "Error: -dbpath, -interval, and -scheduledFor are required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse the interval string into a time.Duration
	intervalDuration, err := time.ParseDuration(*interval)
	if err != nil {
		logger.Error("Invalid interval format", "error", err)
		os.Exit(1)
	}

	// Parse the scheduledFor string into a time.Time
	scheduledForTime, err := time.Parse(time.RFC3339, *scheduledForStr)
	if err != nil {
		logger.Error("Invalid scheduledFor format. Use RFC3339 (e.g., '2025-07-01T10:00:00Z').", "error", err)
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
		Type:         JobTypeDbBackup,
		Payload:      payload,
		ScheduledFor: scheduledForTime, // Use the time from the flag
		Recurrent:    true,
		Interval:     intervalDuration.String(),
	}

	logger.Info("Inserting recurrent backup job into database", "type", newJob.Type, "interval", newJob.Interval, "scheduled_for", newJob.ScheduledFor)

	// Insert the job using the DbQueue interface
	if err := dbConn.InsertJob(newJob); err != nil {
		logger.Error("Failed to insert job", "error", err)
		os.Exit(1)
	}

	logger.Info("Successfully inserted backup job. The server will pick it up on its next cycle.")
}
