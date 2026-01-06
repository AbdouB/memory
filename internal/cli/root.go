// Package cli provides the command-line interface for Memory
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/AbdouB/memory/internal/db"
	"github.com/spf13/cobra"
)

var (
	database   *db.DB
	outputText bool // --text flag for human-readable output (default is JSON for LLMs)
	verbose    bool
)

// rootCmd is the base command
var rootCmd = &cobra.Command{
	Use:   "memory",
	Short: "Epistemic self-awareness framework for AI agents",
	Long: `Memory - Epistemic Self-Awareness for AI Agents

Track what you know and don't know across sessions.

Quick Start:
  memory start "task description"    # Begin session
  memory learned "discovery"         # Log what you learned
  memory uncertain "question"        # Log knowledge gaps
  memory tried "approach" "why"      # Log failed approaches
  memory status                      # See progress
  memory done "summary"              # End session
  memory verify "text"               # Verify stale findings

For more information, visit: https://github.com/AbdouB/memory`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip DB init for help commands
		if cmd.Name() == "help" || cmd.Name() == "version" {
			return nil
		}

		var err error
		database, err = db.Open("")
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if database != nil {
			database.Close()
		}
	},
}

// Execute runs the CLI
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&outputText, "text", false, "Human-readable text output (default is JSON for LLM consumption)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Add version command (core 7 commands are added in quick.go)
	rootCmd.AddCommand(versionCmd)
}

// outputResult outputs the result in the appropriate format
// Default is JSON (for LLMs), use --text for human-readable
func outputResult(result interface{}) {
	if outputText {
		fmt.Printf("%+v\n", result)
	} else {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	}
}

// outputError outputs an error in the appropriate format
// Default is JSON (for LLMs), use --text for human-readable
func outputError(err error) {
	if outputText {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	} else {
		result := map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
		enc := json.NewEncoder(os.Stderr)
		enc.Encode(result)
	}
}

// readStdinJSON reads JSON from stdin
func readStdinJSON(v interface{}) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}
	if len(data) == 0 {
		return fmt.Errorf("no input provided on stdin")
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	return nil
}

// readInputJSON reads JSON from stdin or file
func readInputJSON(input string, v interface{}) error {
	if input == "-" {
		return readStdinJSON(v)
	}

	data, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	return nil
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("memory version 1.0.0 (Go)")
	},
}
