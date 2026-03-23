package cmd

import (
	"io"
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

func workflowServiceWithDeps() (*workflow.Service, error) {
	svc, _, err := workflowServiceWithClosableDeps()
	return svc, err
}

func workflowServiceWithClosableDeps() (*workflow.Service, io.Closer, error) {
	svc, err := workflowService()
	if err != nil {
		return nil, nil, err
	}
	closer := wireWorkflowMVPDeps(svc)
	return svc, closer, nil
}

func newRuntimeTransport(cfg config.Config) *workflow.PersistentProcessTransport {
	return workflow.NewPersistentProcessTransport(cfg.RuntimeBinaryPath)
}
