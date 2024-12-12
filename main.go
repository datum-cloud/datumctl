package main

import (
	"fmt"
	"os"

	"go.datum.net/datumctl/internal/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Printf("failed to execute command: %s\n", err)
		os.Exit(2)
	}
}
