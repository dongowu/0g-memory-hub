package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent [agent-id]",
	Short: "Start interactive agent mode",
	Long: `Start an interactive session to continuously write memories.

Example:
  memory-hub agent alice --tags conversation`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]
		tagsStr, _ := cmd.Flags().GetString("tags")

		tags := parseTags(tagsStr)

		fmt.Printf("Starting agent mode for: %s\n", agentID)
		fmt.Println("Type your memories (Ctrl+C to exit, Ctrl+D to finish input):")
		fmt.Println()

		scanner := bufio.NewScanner(os.Stdin)
		lineNum := 0

		for scanner.Scan() {
			line := scanner.Text()
			lineNum++

			if line == "" {
				continue
			}

			ctx := context.Background()
			mem, err := getHub().Write(ctx, agentID, []byte(line), tags)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}

			fmt.Printf("[%d] Saved: %s\n", lineNum, mem.ID[:8])
		}

		return nil
	},
}

var importCmd = &cobra.Command{
	Use:   "import [agent] [file]",
	Short: "Import memories from file",
	Long: `Import memories from a JSON export file.

Example:
  memory-hub import alice ./alice_backup.json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]
		filePath := args[1]

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		if err := getHub().Import(agentID, data); err != nil {
			return fmt.Errorf("failed to import: %w", err)
		}

		fmt.Printf("Successfully imported memories for agent: %s\n", agentID)
		return nil
	},
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show system information",
	Long:  `Show MemoryHub system information and statistics.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		info := getHub().GetInfo()

		fmt.Println("0G Memory Hub - System Info")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("Data Directory: %s\n", info.IndexInfo.Dir)
		fmt.Printf("Total Agents:   %d\n", info.IndexInfo.Agents)
		fmt.Printf("Total Memories: %d\n", info.IndexInfo.TotalMem)
		fmt.Println()

		if len(info.IndexInfo.Chains) > 0 {
			fmt.Println("Agent Chains:")
			fmt.Println(strings.Repeat("-", 50))
			for agentID, chain := range info.IndexInfo.Chains {
				fmt.Printf("  %s:\n", agentID)
				fmt.Printf("    Head:  %s\n", chain.HeadID[:16])
				fmt.Printf("    Tail:  %s\n", chain.TailID[:16])
				fmt.Printf("    Count: %d\n", chain.Length)
			}
		}

		return nil
	},
}

var verifyCmd = &cobra.Command{
	Use:   "verify [agent]",
	Short: "Verify memory chain integrity",
	Long: `Verify the integrity of an agent's memory chain.

Example:
  memory-hub verify alice`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]

		valid, err := getHub().Verify(agentID)
		if err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}

		if valid {
			fmt.Printf("Chain integrity verified for agent: %s\n", agentID)
		} else {
			fmt.Printf("WARNING: Chain integrity check failed for agent: %s\n", agentID)
		}

		return nil
	},
}

func init() {
	agentCmd.Flags().String("tags", "", "Default tags for memories")

	// Ctrl+C 处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nExiting agent mode...")
		os.Exit(0)
	}()
}
