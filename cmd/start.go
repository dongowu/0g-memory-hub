package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start [wallet]",
	Short: "Start a new session or load existing session",
	Long: `Start a session with your wallet address. If a session already exists,
it will be loaded. Otherwise, a new session will be created.

Examples:
  memory-hub start 0x1234...abcd
  memory-hub start --wallet 0x1234...abcd
  memory-hub start 0x1234...abcd --anchor`,
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wallet := cfg.Wallet

		if wallet == "" && len(args) > 0 {
			wallet = args[0]
		}

		if wallet == "" {
			return fmt.Errorf("wallet address is required")
		}

		session := getHub().GetOrCreateSession(wallet)
		stats := getHub().GetStats(session)

		fmt.Printf("Session loaded successfully!\n\n")
		fmt.Printf("Wallet:    %s\n", stats.Wallet)
		fmt.Printf("Session:   %s\n", session.ID)
		fmt.Printf("Created:   %d messages\n", stats.ContextSize)
		fmt.Printf("Memories:  %d saved\n", stats.MemoryCount)
		fmt.Printf("Tasks:     %d pending / %d total\n", stats.PendingTasks, stats.TaskCount)
		fmt.Printf("Root Hash: %s\n", stats.RootHash)

		return nil
	},
}

func init() {
	startCmd.Flags().StringVar(&cfg.Wallet, "wallet", "", "Wallet address")
}
