package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/dongowu/0g-memory-hub/core"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat [wallet]",
	Short: "Interactive chat mode with memory persistence",
	Long: `Start an interactive chat session with permanent memory.

Features:
  - Cross-session memory persistence
  - Task creation and tracking
  - Automatic memory saving to 0G Storage
  - On-chain anchoring for verification

Commands within chat:
  /task <description> - Create a new task
  /tasks              - List all tasks
  /done <task-id>     - Mark task as done
  /save               - Save current context to 0G
  /anchor             - Anchor memory to blockchain
  /stats              - Show session statistics
  /clear              - Clear current context (saves memory first)
  /quit               - Exit (saves memory first)

Examples:
  memory-hub chat 0x1234...abcd
  memory-hub chat --wallet 0x1234...abcd --model gpt-4`,
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
		ctx := context.Background()

		fmt.Printf("=== 0G Memory Hub Chat ===\n")
		fmt.Printf("Wallet: %s\n", wallet)
		fmt.Printf("Session: %s\n\n", session.ID)
		fmt.Printf("Type /help for commands, /quit to exit\n\n")

		// 加载历史上下文
		context := getHub().GetContext(session)
		if len(context) > 0 {
			fmt.Printf("Loaded %d previous messages\n\n", len(context))
			for _, msg := range context {
				if len(msg.Content) > 200 {
					fmt.Printf("[%s] %s... (truncated)\n", msg.Role, msg.Content[:200])
				} else {
					fmt.Printf("[%s] %s\n", msg.Role, msg.Content)
				}
			}
			fmt.Println()
		}

		scanner := bufio.NewScanner(os.Stdin)

		// 设置信号处理
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// 显示提示符
		fmt.Print("> ")

		for scanner.Scan() {
			select {
			case <-sigChan:
				fmt.Println("\nSaving session...")
				saveAndExit(ctx, session)
				return nil
			default:
			}

			input := strings.TrimSpace(scanner.Text())

			// 处理命令
			if strings.HasPrefix(input, "/") {
				if err := handleCommand(ctx, input, session, cmd); err != nil {
					fmt.Printf("Error: %v\n", err)
				}
				fmt.Print("> ")
				continue
			}

			if input == "" {
				fmt.Print("> ")
				continue
			}

			// 添加用户消息
			getHub().AddMessage(session, "user", input)

			// 模拟 AI 响应 (在真实场景中调用 AI API)
			response := simulateAIResponse(input, session)

			// 添加助手消息
			getHub().AddMessage(session, "assistant", response)

			fmt.Printf("Assistant: %s\n\n", response)
			fmt.Print("> ")

			// 检查是否需要自动保存
			if len(getHub().GetContext(session)) >= 50 {
				fmt.Println("[Auto-saving context...]")
				mem, err := getHub().SaveMemory(ctx, session)
				if err != nil {
					fmt.Printf("Save error: %v\n", err)
				} else {
					fmt.Printf("[Saved: %s, CID: %s...]\n\n", mem.ID, mem.CID[:16])
				}
			}
		}

		// 保存会话
		saveAndExit(ctx, session)

		return nil
	},
}

func handleCommand(ctx context.Context, input string, session *core.Session, cmd *cobra.Command) error {
	parts := strings.SplitN(input, " ", 2)
	command := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	switch command {
	case "/help":
		fmt.Println(`Available commands:
  /task <description>  - Create a new task
  /tasks               - List all tasks
  /done <task-id>      - Mark task as done
  /save                - Save current context to 0G Storage
  /anchor              - Anchor memory to blockchain
  /stats               - Show session statistics
  /clear               - Clear current context (saves memory first)
  /quit                - Exit (saves memory first)`)

	case "/task":
		if args == "" {
			return fmt.Errorf("task description required: /task <description>")
		}
		task := getHub().AddTask(session, args, "once")
		fmt.Printf("Task created: %s\n", task.ID)

	case "/tasks":
		tasks := getHub().ListTasks(session)
		if len(tasks) == 0 {
			fmt.Println("No tasks")
			return nil
		}
		fmt.Println("Tasks:")
		for _, t := range tasks {
			status := "○"
			if t.Status == "done" {
				status = "●"
			}
			fmt.Printf("  [%s] %s (%s)\n", status, t.Description, t.ID)
		}

	case "/done":
		if args == "" {
			return fmt.Errorf("task ID required: /done <task-id>")
		}
		if err := getHub().CompleteTask(session, args); err != nil {
			return err
		}
		fmt.Printf("Task %s marked as done\n", args)

	case "/save":
		mem, err := getHub().SaveMemory(ctx, session)
		if err != nil {
			return fmt.Errorf("failed to save: %w", err)
		}
		fmt.Printf("Memory saved: %s\n", mem.ID)
		fmt.Printf("CID: %s\n", mem.CID)

	case "/anchor":
		if cfg.PrivateKey == "" {
			fmt.Println("No private key configured. Use --key flag or KEY env var.")
			return nil
		}
		result, err := getHub().AnchorToChain(ctx, session)
		if err != nil {
			return fmt.Errorf("failed to anchor: %w", err)
		}
		fmt.Printf("Anchored to chain!\n")
		fmt.Printf("TxHash: %s\n", result.TxHash)

	case "/stats":
		stats := getHub().GetStats(session)
		fmt.Printf("Session Statistics:\n")
		fmt.Printf("  Messages: %d\n", stats.ContextSize)
		fmt.Printf("  Memories: %d\n", stats.MemoryCount)
		fmt.Printf("  Tasks:    %d pending / %d total\n", stats.PendingTasks, stats.TaskCount)
		fmt.Printf("  Root:     %s\n", stats.RootHash)

	case "/clear":
		mem, _ := getHub().SaveMemory(ctx, session)
		if mem != nil {
			fmt.Printf("Saved to memory: %s\n", mem.ID)
		}
		session.ClearContext()
		fmt.Println("Context cleared")

	case "/quit", "/exit":
		saveAndExit(ctx, session)
		return fmt.Errorf("quit signal")

	default:
		fmt.Printf("Unknown command: %s\n", command)
	}

	return nil
}

func saveAndExit(ctx context.Context, session *core.Session) {
	mem, err := getHub().SaveMemory(ctx, session)
	if err != nil {
		fmt.Printf("Warning: failed to save memory: %v\n", err)
	} else {
		fmt.Printf("Session saved: %s\n", mem.ID)
	}

	if err := getHub().SaveSession(session); err != nil {
		fmt.Printf("Warning: failed to save session: %v\n", err)
	}
}

func simulateAIResponse(input string, session *core.Session) string {
	// 简单的模拟响应 - 在真实场景中调用 AI API
	input = strings.ToLower(input)

	if strings.Contains(input, "hello") || strings.Contains(input, "hi") {
		return "Hello! I remember our previous conversations thanks to 0G Memory Hub."
	}

	if strings.Contains(input, "task") {
		return "I've noted that. Would you like me to add it to your task list?"
	}

	if strings.Contains(input, "remember") {
		return "I can recall information from previous sessions thanks to permanent storage on 0G."
	}

	pending := len(getHub().GetPendingTasks(session))
	if pending > 0 {
		return fmt.Sprintf("I've processed that. You have %d pending tasks. Should I help you with them?", pending)
	}

	return "Understood. Your context is being saved for next session."
}

func init() {
	chatCmd.Flags().StringVar(&cfg.Wallet, "wallet", "", "Wallet address")
}
