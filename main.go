package main

import (
	"os"

	"github.com/AbdouB/memory/internal/cli"
)

var Version = "dev"

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
