package main

import (
	"fmt"
	"os"

	sqlitebackup "github.com/caasmo/restinpieces-sqlite-backup"
	"github.com/pelletier/go-toml/v2"
)

func main() {
	// Get the default configuration from the package
	blueprint := sqlitebackup.GenerateBlueprintConfig()

	// Marshal the blueprint struct to a TOML-formatted byte slice.
	// The custom Duration type ensures durations are marshaled as strings.
	tomlBytes, err := toml.Marshal(blueprint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding TOML: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("# Default configuration for the sqlite_backup job.")
	fmt.Println("# Save this content and use the restinpieces framework to securely store it")
	fmt.Println("# under the scope 'sqlite_backup'.")
	fmt.Println()
	fmt.Println(string(tomlBytes))
}