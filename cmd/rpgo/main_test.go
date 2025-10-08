package main

import (
	"bytes"
	"testing"
)

func TestRootCommand(t *testing.T) {
	cmd := rootCmd

	if cmd == nil {
		t.Fatal("Expected root command to be created")
	}

	if cmd.Use != "rpgo" {
		t.Errorf("Expected root command use to be 'rpgo', got %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Expected root command to have a short description")
	}

	if cmd.Long == "" {
		t.Error("Expected root command to have a long description")
	}
}

func TestRootCommand_Execute(t *testing.T) {
	// Test that the root command can be executed without arguments
	cmd := rootCmd
	cmd.SetArgs([]string{})

	// Capture output
	var buf bytes.Buffer
	cmd.SetOutput(&buf)

	// Execute the command
	err := cmd.Execute()

	// Should show help/usage
	if err != nil {
		t.Errorf("Expected no error for root command execution, got %v", err)
	}

	// Check that help is shown
	output := buf.String()
	if output == "" {
		t.Error("Expected root command to show help/usage")
	}
}

func TestRootCommand_Help(t *testing.T) {
	cmd := rootCmd
	cmd.SetArgs([]string{"--help"})

	var buf bytes.Buffer
	cmd.SetOutput(&buf)

	err := cmd.Execute()

	if err != nil {
		t.Errorf("Expected no error for help command, got %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected help command to show help text")
	}
}

func TestCommandSubcommands(t *testing.T) {
	// Test that all expected commands are registered
	expectedCommands := []string{
		"calculate",
		"compare",
		"optimize",
		"plan-roth",
		"analyze-survivor",
		"fers-monte-carlo",
	}

	cmd := rootCmd.Commands()
	for _, expectedCmd := range expectedCommands {
		found := false
		for _, c := range cmd {
			if c.Name() == expectedCmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected command '%s' to be registered with root command", expectedCmd)
		}
	}
}

func TestFileExists(t *testing.T) {
	// Test with existing file (use a file we know exists in the project)
	if !fileExists("/Users/robertgehrsitz/Code/rpgo/README.md") {
		t.Error("Expected README.md to exist")
	}

	// Test with non-existing file
	if fileExists("non_existing_file.txt") {
		t.Error("Expected non_existing_file.txt to not exist")
	}
}

func TestRootCommandFlags(t *testing.T) {
	cmd := rootCmd

	// Test help flag (should exist by default in cobra)
	helpFlag := cmd.Flag("help")
	if helpFlag == nil {
		t.Error("Expected help flag to exist on root command")
	}
}

func TestRootCommand_InvalidCommand(t *testing.T) {
	cmd := rootCmd
	cmd.SetArgs([]string{"invalid-command"})

	var buf bytes.Buffer
	cmd.SetOutput(&buf)

	err := cmd.Execute()

	// Should show error for invalid command
	if err == nil {
		t.Error("Expected error for invalid command")
	}
}

func TestRootCommand_InvalidFlag(t *testing.T) {
	cmd := rootCmd
	cmd.SetArgs([]string{"--invalid-flag"})

	var buf bytes.Buffer
	cmd.SetOutput(&buf)

	err := cmd.Execute()

	// Should show error for invalid flag
	if err == nil {
		t.Error("Expected error for invalid flag")
	}
}
