package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	sqlitebackup "github.com/caasmo/restinpieces-sqlite-backup"
	"github.com/pelletier/go-toml/v2"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	outputFileFlag := flag.String("output", "backup.blueprint.toml", "Output file path for the blueprint TOML configuration")
	flag.StringVar(outputFileFlag, "o", "backup.blueprint.toml", "Output file path (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generates a blueprint backup TOML configuration file with example values.\n")
		fmt.Fprintf(os.Stderr, "Remember to replace placeholder values.\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	logger.Info("Generating backup blueprint configuration...")
	blueprintCfg := sqlitebackup.GenerateBlueprintConfig()

	logger.Info("Marshalling configuration to TOML...")
	tomlBytes, err := toml.Marshal(blueprintCfg)
	if err != nil {
		logger.Error("Failed to marshal blueprint config to TOML", "error", err)
		os.Exit(1)
	}

	logger.Info("Writing blueprint configuration", "path", *outputFileFlag)
	err = os.WriteFile(*outputFileFlag, tomlBytes, 0644)
	if err != nil {
		logger.Error("Failed to write blueprint file", "path", *outputFileFlag, "error", err)
		os.Exit(1)
	}

	logger.Info("Successfully created blueprint configuration file", "path", *outputFileFlag)
}