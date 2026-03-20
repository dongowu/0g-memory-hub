package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dongowu/0g-memory-hub/core"
	"github.com/dongowu/0g-memory-hub/sdk"
	"github.com/spf13/cobra"
)

var queryCmd = &cobra.Command{
	Use:   "query [agent]",
	Short: "Query memory history",
	Long: `Query memory history with various filters.

Examples:
  memory-hub query alice --last 50
  memory-hub query alice --since 1700000000 --until 1700100000
  memory-hub query alice --tag learning
  memory-hub query alice --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]

		lastN, _ := cmd.Flags().GetInt("last")
		sinceStr, _ := cmd.Flags().GetString("since")
		untilStr, _ := cmd.Flags().GetString("until")
		tag, _ := cmd.Flags().GetString("tag")
		asJSON, _ := cmd.Flags().GetBool("json")
		outputPath, _ := cmd.Flags().GetString("output")

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

		memories, err := getHub().Query(agentID, opts)
		if err != nil {
			return fmt.Errorf("failed to query: %w", err)
		}

		if len(memories) == 0 {
			fmt.Printf("No memories found for agent: %s\n", agentID)
			return nil
		}

		if asJSON {
			return printJSON(memories, outputPath)
		}

		fmt.Printf("Found %d memories for agent: %s\n\n", len(memories), agentID)
		fmt.Printf("%-12s %-36s %-10s %-8s %s\n", "TIMESTAMP", "ID", "SIZE", "STATUS", "TAGS")
		fmt.Println(strings.Repeat("-", 100))

		for _, mem := range memories {
			timestamp := time.Unix(mem.Timestamp, 0).Format("2006-01-02 15:04")
			tags := strings.Join(mem.Tags, ",")
			if len(tags) > 20 {
				tags = tags[:17] + "..."
			}
			fmt.Printf("%-12s %-36s %-10d %-8s %s\n",
				timestamp, mem.ID[:36], mem.Size, mem.Status, tags)
		}

		return nil
	},
}

func parseTime(s string) (time.Time, error) {
	var t time.Time
	var err error

	if len(s) <= 10 {
		var unix int64
		_, err = fmt.Sscanf(s, "%d", &unix)
		if err == nil {
			return time.Unix(unix, 0), nil
		}
	}

	t, err = time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}

	t, err = time.Parse("2006-01-02", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("cannot parse time: %s", s)
}

func printJSON(memories []*core.Memory, outputPath string) error {
	entries := make([]*core.MemoryEntry, len(memories))
	for i, mem := range memories {
		entries[i] = mem.ToEntry()
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	if outputPath != "" {
		return os.WriteFile(outputPath, data, 0644)
	}

	fmt.Println(string(data))
	return nil
}

func init() {
	queryCmd.Flags().Int("last", 0, "Number of recent memories")
	queryCmd.Flags().String("since", "", "Start time")
	queryCmd.Flags().String("until", "", "End time")
	queryCmd.Flags().String("tag", "", "Filter by tag")
	queryCmd.Flags().Bool("json", false, "Output as JSON")
	queryCmd.Flags().StringP("output", "o", "", "Output file path")
}
