package cmd

import (
	"fmt"

	"github.com/dongowu/0g-memory-hub/sdk"
	"github.com/spf13/cobra"
)

var replayCmd = &cobra.Command{
	Use:   "replay [agent]",
	Short: "Replay memory history",
	Long: `Replay memory history as a conversation.

Examples:
  memory-hub replay alice --last 20
  memory-hub replay alice --since 1700000000
  memory-hub replay alice --tag conversation --format chat`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]

		lastN, _ := cmd.Flags().GetInt("last")
		sinceStr, _ := cmd.Flags().GetString("since")
		untilStr, _ := cmd.Flags().GetString("until")
		tag, _ := cmd.Flags().GetString("tag")
		format, _ := cmd.Flags().GetString("format")

		opts := &sdk.QueryOptions{
			LastN: int64(lastN),
			Tag:   tag,
		}

		if sinceStr != "" {
			t, err := parseTime(sinceStr)
			if err != nil {
				return fmt.Errorf("invalid --since: %w", err)
			}
			opts.Since = t.Unix()
		}

		if untilStr != "" {
			t, err := parseTime(untilStr)
			if err != nil {
				return fmt.Errorf("invalid --until: %w", err)
			}
			opts.Until = t.Unix()
		}

		lines, err := getHub().Replay(agentID, opts)
		if err != nil {
			return fmt.Errorf("failed to replay: %w", err)
		}

		if len(lines) == 0 {
			fmt.Printf("No memories to replay for agent: %s\n", agentID)
			return nil
		}

		switch format {
		case "chat":
			for _, line := range lines {
				fmt.Printf("[%s] %s\n", agentID, line)
			}
		default:
			for _, line := range lines {
				fmt.Println(line)
			}
		}

		return nil
	},
}

func init() {
	replayCmd.Flags().Int("last", 20, "Number of recent memories")
	replayCmd.Flags().String("since", "", "Start time")
	replayCmd.Flags().String("until", "", "End time")
	replayCmd.Flags().String("tag", "", "Filter by tag")
	replayCmd.Flags().String("format", "simple", "Output format (simple, chat)")
}
