package cmd

import "testing"

func TestWorkflowCommandIncludesVerifyCommand(t *testing.T) {
	t.Parallel()

	foundWorkflow := false
	foundVerify := false

	for _, c := range rootCmd.Commands() {
		if c.Name() != "workflow" {
			continue
		}
		foundWorkflow = true
		for _, sub := range c.Commands() {
			if sub.Name() == "verify" {
				foundVerify = true
				break
			}
		}
		break
	}

	if !foundWorkflow {
		t.Fatal("workflow command not registered on root command")
	}
	if !foundVerify {
		t.Fatal("verify command not registered on workflow command")
	}
}
