package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Task management commands",
	Long: `Manage tasks across sessions.

Subcommands:
  list    - List all tasks
  add     - Add a new task
  done    - Mark task as done
  pending - Show pending tasks only

Examples:
  memory-hub task list 0x1234...abcd
  memory-hub task add 0x1234...abcd "Review PR #42"
  memory-hub task done 0x1234...abcd task_12345`,
}

var taskListCmd = &cobra.Command{
	Use:   "list [wallet]",
	Short: "List all tasks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wallet := args[0]
		session := getHub().GetOrCreateSession(wallet)

		tasks := getHub().ListTasks(session)
		if len(tasks) == 0 {
			fmt.Printf("No tasks for wallet: %s\n", wallet)
			return nil
		}

		pending := 0
		done := 0
		for _, t := range tasks {
			if t.Status == "pending" {
				pending++
			} else {
				done++
			}
		}

		fmt.Printf("Tasks for %s: %d pending, %d done\n\n", wallet, pending, done)
		fmt.Printf("%-8s %-10s %-12s %s\n", "STATUS", "ID", "DUE DATE", "DESCRIPTION")
		fmt.Println(strings.Repeat("-", 80))

		for _, t := range tasks {
			status := "pending"
			if t.Status == "done" {
				status = "done"
			}

			due := "-"
			if t.DueDate > 0 {
				due = time.Unix(t.DueDate, 0).Format("2006-01-02")
			}

			desc := t.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}

			fmt.Printf("%-8s %-10s %-12s %s\n", status, t.ID, due, desc)
		}

		return nil
	},
}

var taskAddCmd = &cobra.Command{
	Use:   "add [wallet] [description]",
	Short: "Add a new task",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		wallet := args[0]
		description := args[1]

		session := getHub().GetOrCreateSession(wallet)
		task := getHub().AddTask(session, description, "once")

		// 保存会话
		if err := getHub().SaveSession(session); err != nil {
			return fmt.Errorf("failed to save session: %w", err)
		}

		fmt.Printf("Task created: %s\n", task.ID)
		fmt.Printf("Description: %s\n", task.Description)

		return nil
	},
}

var taskDoneCmd = &cobra.Command{
	Use:   "done [wallet] [task-id]",
	Short: "Mark task as done",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		wallet := args[0]
		taskID := args[1]

		session := getHub().GetOrCreateSession(wallet)

		if err := getHub().CompleteTask(session, taskID); err != nil {
			return err
		}

		// 保存会话
		if err := getHub().SaveSession(session); err != nil {
			return fmt.Errorf("failed to save session: %w", err)
		}

		fmt.Printf("Task %s marked as done\n", taskID)

		return nil
	},
}

var taskPendingCmd = &cobra.Command{
	Use:   "pending [wallet]",
	Short: "Show pending tasks only",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wallet := args[0]
		session := getHub().GetOrCreateSession(wallet)

		tasks := getHub().GetPendingTasks(session)
		if len(tasks) == 0 {
			fmt.Printf("No pending tasks for wallet: %s\n", wallet)
			return nil
		}

		fmt.Printf("Pending tasks for %s:\n\n", wallet)

		for _, t := range tasks {
			due := ""
			if t.DueDate > 0 {
				due = fmt.Sprintf(" (due: %s)", time.Unix(t.DueDate, 0).Format("2006-01-02"))
			}

			freq := ""
			if t.Frequency != "" && t.Frequency != "once" {
				freq = fmt.Sprintf(" [%s]", t.Frequency)
			}

			fmt.Printf("[%s]%s%s\n", t.ID, due, freq)
			fmt.Printf("    %s\n\n", t.Description)
		}

		return nil
	},
}

var taskSaveCmd = &cobra.Command{
	Use:   "save [wallet]",
	Short: "Save task list to 0G",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		wallet := args[0]
		ctx := context.Background()

		session := getHub().GetOrCreateSession(wallet)

		mem, err := getHub().SaveMemory(ctx, session)
		if err != nil {
			return fmt.Errorf("failed to save: %w", err)
		}

		fmt.Printf("Tasks saved to 0G Storage!\n")
		fmt.Printf("Memory ID: %s\n", mem.ID)
		fmt.Printf("CID: %s\n", mem.CID)

		if cfg.PrivateKey != "" {
			result, err := getHub().AnchorToChain(ctx, session)
			if err != nil {
				fmt.Printf("Warning: failed to anchor: %v\n", err)
			} else {
				fmt.Printf("TxHash: %s\n", result.TxHash)
			}
		}

		return nil
	},
}

func init() {
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskAddCmd)
	taskCmd.AddCommand(taskDoneCmd)
	taskCmd.AddCommand(taskPendingCmd)
	taskCmd.AddCommand(taskSaveCmd)

	// 为 list, pending 命令添加输出选项
	taskListCmd.Flags().BoolP("json", "j", false, "Output as JSON")
	taskPendingCmd.Flags().BoolP("json", "j", false, "Output as JSON")

	// 为 add 命令添加 frequency 选项
	taskAddCmd.Flags().StringP("frequency", "f", "once", "Task frequency (once, daily, weekly)")

	// 为 pending 命令添加 due-date 选项
	taskPendingCmd.Flags().StringP("due", "d", "", "Due date (YYYY-MM-DD)")

	// 允许未定义子命令时显示帮助
	if len(os.Args) > 1 && os.Args[1] == "task" && len(os.Args) < 3 {
		taskCmd.Help()
		os.Exit(0)
	}
}
