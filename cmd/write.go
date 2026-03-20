package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var writeCmd = &cobra.Command{
	Use:   "write [agent] [content]",
	Short: "Write a memory to 0G Storage",
	Long:  `Write a memory entry for an agent and upload it to 0G Storage.

Examples:
  memory-hub write alice "Hello, world!"
  memory-hub write bob --file ./memory.json --tags learning,conversation
  memory-hub write carol --tags trade --content "buy btc"`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentID := args[0]

		var content []byte
		var err error

		// 获取内容
		if filePath != "" {
			content, err = os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
		} else if len(args) > 1 {
			content = []byte(args[1])
		} else {
			// 从 stdin 读取
			fmt.Println("Enter memory content (Ctrl+D to finish):")
			data := make([]byte, 0)
			buf := make([]byte, 1024)
			for {
				n, err := os.Stdin.Read(buf)
				if n > 0 {
					data = append(data, buf[:n]...)
				}
				if err != nil {
					break
				}
			}
			content = data
		}

		// 解析标签
		tags := parseTags(tagStr)

		// 写入
		ctx := context.Background()
		mem, err := getHub().Write(ctx, agentID, content, tags)
		if err != nil {
			return fmt.Errorf("failed to write memory: %w", err)
		}

		// 输出结果
		fmt.Println("\n Memory written successfully!")
		fmt.Printf("   ID:       %s\n", mem.ID)
		fmt.Printf("   Agent:    %s\n", mem.AgentID)
		fmt.Printf("   CID:      %s\n", mem.CID)
		fmt.Printf("   TxHash:   %s\n", mem.TxHash)
		fmt.Printf("   Size:     %d bytes\n", mem.Size)
		fmt.Printf("   Status:   %s\n", mem.Status)
		if len(mem.Tags) > 0 {
			fmt.Printf("   Tags:     %s\n", strings.Join(mem.Tags, ", "))
		}

		return nil
	},
}

var (
	filePath string
	tagStr   string
)

func init() {
	writeCmd.Flags().StringVarP(&filePath, "file", "f", "", "Read content from file")
	writeCmd.Flags().StringVarP(&tagStr, "tags", "t", "", "Comma-separated tags")
}

func parseTags(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}
