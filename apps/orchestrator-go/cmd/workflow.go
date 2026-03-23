package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/config"
	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/ogchain"
	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/ogstorage"
	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/openclaw"
	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/workflow"
	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Workflow lifecycle commands",
}

var workflowStartCmd = &cobra.Command{
	Use:   "start [workflow-id]",
	Short: "Start a workflow run",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		svc, err := workflowService()
		if err != nil {
			return err
		}

		workflowID := ""
		if len(args) == 1 {
			workflowID = args[0]
		}

		meta, err := svc.Start(workflowID)
		if err != nil {
			return err
		}

		fmt.Printf("workflow_id=%s status=%s latest_step=%d\n", meta.WorkflowID, meta.Status, meta.LatestStep)
		return nil
	},
}

var workflowStepCmd = &cobra.Command{
	Use:   "step [workflow-id]",
	Short: "Record a workflow step",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		eventType, _ := cmd.Flags().GetString("event-type")
		actor, _ := cmd.Flags().GetString("actor")
		payload, _ := cmd.Flags().GetString("payload")

		svc, err := workflowService()
		if err != nil {
			return err
		}
		wireWorkflowMVPDeps(svc)

		evt := openclaw.NormalizeStep(openclaw.StepInput{
			WorkflowID: args[0],
			EventType:  eventType,
			Actor:      actor,
			Payload:    payload,
		})

		meta, err := svc.Step(context.Background(), args[0], evt)
		if err != nil {
			return err
		}

		fmt.Printf("workflow_id=%s status=%s latest_step=%d latest_root=%s latest_cid=%s latest_tx_hash=%s\n",
			meta.WorkflowID, meta.Status, meta.LatestStep, meta.LatestRoot, meta.LatestCID, meta.LatestTxHash)
		return nil
	},
}

type workflowStorageAdapter struct {
	client ogstorage.Client
}

func (a workflowStorageAdapter) UploadCheckpoint(ctx context.Context, payload []byte) (string, string, error) {
	result, err := a.client.UploadCheckpoint(ctx, payload)
	if err != nil {
		return "", "", err
	}
	return result.Key, result.TxHash, nil
}

func (a workflowStorageAdapter) DownloadCheckpoint(ctx context.Context, key string) ([]byte, error) {
	return a.client.DownloadCheckpoint(ctx, key)
}

func (a workflowStorageAdapter) CheckReadiness(ctx context.Context) error {
	if checker, ok := a.client.(interface{ CheckReadiness(context.Context) error }); ok {
		return checker.CheckReadiness(ctx)
	}
	return nil
}

type workflowAnchorAdapter struct {
	client ogchain.Client
}

func (a workflowAnchorAdapter) AnchorCheckpoint(ctx context.Context, in workflow.AnchorInput) (string, error) {
	result, err := a.client.AnchorCheckpoint(ctx, ogchain.AnchorInput{
		WorkflowID: in.WorkflowID,
		StepIndex:  in.StepIndex,
		RootHash:   in.RootHash,
		CIDHash:    in.CIDHash,
	})
	if err != nil {
		return "", err
	}
	return result.TxHash, nil
}

func (a workflowAnchorAdapter) CheckReadiness(ctx context.Context) error {
	if checker, ok := a.client.(interface{ CheckReadiness(context.Context) error }); ok {
		return checker.CheckReadiness(ctx)
	}
	return nil
}

func wireWorkflowMVPDeps(svc *workflow.Service) io.Closer {
	cfg := config.Load()
	runtimeTransport := newRuntimeTransport(cfg)
	runtimeClient := workflow.NewRuntimeClient(runtimeTransport)
	storageClient := ogstorage.NewSDKClient(ogstorage.SDKConfig{
		IndexerRPCURL:    cfg.StorageRPCURL,
		BlockchainRPCURL: cfg.ChainRPCURL,
		PrivateKey:       cfg.ChainPrivateKey,
		ChainID:          cfg.ChainID,
	}, nil)
	chainClient := ogchain.NewJSONRPCClient(
		cfg.ChainRPCURL,
		cfg.ChainPrivateKey,
		cfg.ChainContractAddress,
		cfg.ChainID,
		nil,
	)

	svc.SetRuntime(runtimeClient)
	svc.SetStorage(workflowStorageAdapter{client: storageClient})
	svc.SetAnchor(workflowAnchorAdapter{client: chainClient})
	return runtimeTransport
}

var workflowResumeCmd = &cobra.Command{
	Use:   "resume [workflow-id]",
	Short: "Resume a workflow run",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		svc, err := workflowServiceWithDeps()
		if err != nil {
			return err
		}

		meta, err := svc.ResumeWithContext(context.Background(), args[0])
		if err != nil {
			return err
		}

		fmt.Printf("workflow_id=%s status=%s latest_step=%d\n", meta.WorkflowID, meta.Status, meta.LatestStep)
		return nil
	},
}

var workflowReplayCmd = &cobra.Command{
	Use:   "replay [workflow-id]",
	Short: "Replay workflow trace",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		svc, err := workflowService()
		if err != nil {
			return err
		}

		lines, err := svc.Replay(args[0])
		if err != nil {
			return err
		}
		for _, line := range lines {
			fmt.Println(line)
		}
		return nil
	},
}

var workflowStatusCmd = &cobra.Command{
	Use:   "status [workflow-id]",
	Short: "Show workflow metadata",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		svc, err := workflowService()
		if err != nil {
			return err
		}

		meta, err := svc.Status(args[0])
		if err != nil {
			return err
		}

		fmt.Printf("workflow_id=%s status=%s latest_step=%d latest_root=%s latest_cid=%s latest_tx_hash=%s\n",
			meta.WorkflowID, meta.Status, meta.LatestStep, meta.LatestRoot, meta.LatestCID, meta.LatestTxHash)
		return nil
	},
}

func init() {
	workflowStepCmd.Flags().String("event-type", "task_event", "Normalized event type")
	workflowStepCmd.Flags().String("actor", "openclaw", "Actor producing this event")
	workflowStepCmd.Flags().String("payload", "{}", "Event payload JSON string")

	workflowCmd.AddCommand(
		workflowStartCmd,
		workflowStepCmd,
		workflowResumeCmd,
		workflowReplayCmd,
		workflowStatusCmd,
	)

	rootCmd.AddCommand(workflowCmd)
}
