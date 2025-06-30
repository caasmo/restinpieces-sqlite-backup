
package main

import (
	"fmt"
	"os"

	sqlitebackup "github.com.com/caasmo/restinpieces-sqlite-backup"
	"github.com/pelletier/go-toml/v2"
)

const outputFilename = "config.toml.blueprint"

func main() {
	// Get the default configuration from the package
	blueprint := sqlitebackup.GenerateBlueprintConfig()

	// Marshal the blueprint struct to a TOML-formatted byte slice.
	tomlBytes, err := toml.Marshal(blueprint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding TOML: %v\n", err)
		os.Exit(1)
	}

	// Write the content to the blueprint file.
	err = os.WriteFile(outputFilename, tomlBytes, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to file %s: %v\n", outputFilename, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully created blueprint configuration file: %s\n", outputFilename)
	fmt.Println("You can now edit this file and use it to configure the backup job.")
}
