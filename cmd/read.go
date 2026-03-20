package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dongowu/0g-memory-hub/core"
	"github.com/dongowu/0g-memory-hub/sdk"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read [agent] [memory-id]",
	Short: "Read a memory entry",
	Long: `Read a specific memory entry by ID or read the latest memories.

Examples:
  memory-hub read alice
  memory-hub read alice --last 10
  memory-hub read alice abc123-456 --show-content`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]

		ctx := context.Background()
		showContent, _ := cmd.Flags().GetBool("show-content")
		lastN, _ := cmd.Flags().GetInt("last")

		var outputPath string
		if of, _ := cmd.Flags().GetString("output"); of != "" {
			outputPath = of
		}

		if len(args) > 1 && args[1] != "" {
			mem, err := getHub().Read(ctx, agentID, args[1])
			if err != nil {
				return fmt.Errorf("failed to read memory: %w", err)
			}

			if outputPath != "" {
				os.WriteFile(outputPath, mem.Content, 0644)
				fmt.Printf("Content written to: %s\n", outputPath)
			} else {
				printMemory(mem, showContent)
			}
		} else {
			memories, err := getHub().Query(agentID, &sdk.QueryOptions{LastN: int64(lastN)})
			if err != nil {
				return fmt.Errorf("failed to query: %w", err)
			}

			if len(memories) == 0 {
				fmt.Printf("No memories found for agent: %s\n", agentID)
				return nil
			}

			for i, mem := range memories {
				if i > 0 {
					fmt.Println("---")
				}
				printMemory(mem, showContent)
			}
		}

		return nil
	},
}

func printMemory(mem *core.Memory, showContent bool) {
	timestamp := time.Unix(mem.Timestamp, 0).Format(time.RFC3339)

	fmt.Printf("ID:        %s\n", mem.ID)
	fmt.Printf("Agent:     %s\n", mem.AgentID)
	fmt.Printf("Timestamp: %s\n", timestamp)
	fmt.Printf("CID:       %s\n", mem.CID)
	fmt.Printf("Hash:      %s\n", mem.Hash)
	fmt.Printf("Size:      %d bytes\n", mem.Size)
	fmt.Printf("Status:    %s\n", mem.Status)

	if len(mem.Tags) > 0 {
		fmt.Printf("Tags:      %s\n", strings.Join(mem.Tags, ", "))
	}

	if showContent && len(mem.Content) > 0 {
		fmt.Printf("\nContent:\n%s\n", string(mem.Content))
	}
}

func init() {
	readCmd.Flags().Bool("show-content", false, "Show memory content")
	readCmd.Flags().Int("last", 1, "Number of recent memories to read")
	readCmd.Flags().StringP("output", "o", "", "Output file path")
}
