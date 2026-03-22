package main

import (
	"fmt"
	"os"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
