package cmd

import (
	"path/filepath"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/config"
	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/workflow"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "orchestrator-go",
	Short: "0G OpenClaw orchestrator skeleton",
}

func Execute() error {
	return rootCmd.Execute()
}

func workflowService() (*workflow.Service, error) {
	cfg := config.Load()
	storePath := filepath.Join(cfg.DataDir, "workflows.json")
	store, err := workflow.NewFileStore(storePath)
	if err != nil {
		return nil, err
	}
	return workflow.NewService(store), nil
}
