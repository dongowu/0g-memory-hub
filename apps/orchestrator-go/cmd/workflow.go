package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/config"
	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/ogchain"
	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/ogkv"
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
	store  *workflow.LocalFileStorage
}

func (a workflowStorageAdapter) UploadCheckpoint(ctx context.Context, payload []byte) (string, string, error) {
	if a.store != nil {
		return a.store.UploadCheckpoint(ctx, payload)
	}
	result, err := a.client.UploadCheckpoint(ctx, payload)
	if err != nil {
		return "", "", err
	}
	return result.Key, result.TxHash, nil
}

func (a workflowStorageAdapter) DownloadCheckpoint(ctx context.Context, key string) ([]byte, error) {
	if a.store != nil {
		return a.store.DownloadCheckpoint(ctx, key)
	}
	return a.client.DownloadCheckpoint(ctx, key)
}

func (a workflowStorageAdapter) CheckReadiness(ctx context.Context) error {
	if a.store != nil {
		return a.store.CheckReadiness(ctx)
	}
	if checker, ok := a.client.(interface{ CheckReadiness(context.Context) error }); ok {
		return checker.CheckReadiness(ctx)
	}
	return nil
}

type workflowAnchorAdapter struct {
	client ogchain.Client
	anchor *workflow.LocalAnchor
}

func (a workflowAnchorAdapter) AnchorCheckpoint(ctx context.Context, in workflow.AnchorInput) (string, error) {
	if a.anchor != nil {
		return a.anchor.AnchorCheckpoint(ctx, in)
	}
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

func (a workflowAnchorAdapter) GetLatestCheckpoint(ctx context.Context, workflowID string) (workflow.AnchorLatestCheckpoint, error) {
	if a.anchor != nil {
		return a.anchor.GetLatestCheckpoint(ctx, workflowID)
	}
	result, err := a.client.GetLatestCheckpoint(ctx, workflowID)
	if err != nil {
		return workflow.AnchorLatestCheckpoint{}, err
	}
	return workflow.AnchorLatestCheckpoint{
		StepIndex: result.StepIndex,
		RootHash:  result.RootHash,
		CIDHash:   result.CIDHash,
		Timestamp: result.Timestamp,
		Submitter: result.Submitter,
	}, nil
}

func (a workflowAnchorAdapter) CheckReadiness(ctx context.Context) error {
	if a.anchor != nil {
		return nil // local anchor is always ready
	}
	if checker, ok := a.client.(interface{ CheckReadiness(context.Context) error }); ok {
		return checker.CheckReadiness(ctx)
	}
	return nil
}

type workflowKVAdapter struct {
	client *ogkv.Client
}

func (a workflowKVAdapter) PutCheckpoint(ctx context.Context, summary workflow.KVCheckpointSummary) error {
	return a.client.PutCheckpoint(ctx, ogkv.CheckpointSummary{
		WorkflowID: summary.WorkflowID,
		StepIndex:  summary.StepIndex,
		RootHash:   summary.RootHash,
		CID:        summary.CID,
		TxHash:     summary.TxHash,
		Timestamp:  summary.Timestamp,
	})
}

func (a workflowKVAdapter) GetLatestCheckpoint(ctx context.Context, workflowID string) (*workflow.KVCheckpointSummary, error) {
	s, err := a.client.GetLatestCheckpoint(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	return &workflow.KVCheckpointSummary{
		WorkflowID: s.WorkflowID,
		StepIndex:  s.StepIndex,
		RootHash:   s.RootHash,
		CID:        s.CID,
		TxHash:     s.TxHash,
		Timestamp:  s.Timestamp,
	}, nil
}

func (a workflowKVAdapter) CheckReadiness(ctx context.Context) error {
	return a.client.CheckReadiness(ctx)
}

func wireWorkflowMVPDeps(svc *workflow.Service) io.Closer {
	cfg := config.Load()
	runtimeTransport := newRuntimeTransport(cfg)
	runtimeClient := workflow.NewRuntimeClient(runtimeTransport)

	// Use local file-based storage/anchor when private key is not configured
	if cfg.ChainPrivateKey == "" {
		localStore, err := workflow.NewLocalFileStorage(filepath.Join(cfg.DataDir, "local-storage"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not create local storage: %v\n", err)
		} else {
			svc.SetStorage(workflowStorageAdapter{store: localStore})
		}
		localAnchor, err := workflow.NewLocalAnchor(filepath.Join(cfg.DataDir, "local-storage"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not create local anchor: %v\n", err)
		} else {
			svc.SetAnchor(workflowAnchorAdapter{anchor: localAnchor})
		}
	} else {
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
		svc.SetStorage(workflowStorageAdapter{client: storageClient})
		svc.SetAnchor(workflowAnchorAdapter{client: chainClient})
	}

	svc.SetRuntime(runtimeClient)

	if cfg.KVNodeURL != "" {
		kvClient := ogkv.NewClient(ogkv.Config{
			KVNodeURL:     cfg.KVNodeURL,
			ZgsNodeURL:    cfg.KVZgsNodeURL,
			BlockchainRPC: cfg.ChainRPCURL,
			PrivateKey:    cfg.ChainPrivateKey,
		})
		svc.SetKVStore(workflowKVAdapter{client: kvClient})
	}

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

var workflowVerifyCmd = &cobra.Command{
	Use:   "verify [run-id]",
	Short: "Verify a workflow run and print judge-facing JSON",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, closer, err := workflowServiceWithClosableDeps()
		if err != nil {
			return err
		}
		if closer != nil {
			defer closer.Close()
		}

		verifyResult, err := svc.VerifyRun(context.Background(), args[0])
		if err != nil {
			return err
		}

		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		return encoder.Encode(verifyResult)
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
		workflowVerifyCmd,
	)

	rootCmd.AddCommand(workflowCmd)
}
