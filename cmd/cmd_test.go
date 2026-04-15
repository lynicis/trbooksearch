package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSetVersionInfo(t *testing.T) {
	origVersion := appVersion
	t.Cleanup(func() {
		appVersion = origVersion
	})

	SetVersionInfo("1.2.3")

	if appVersion != "1.2.3" {
		t.Errorf("appVersion = %q, want %q", appVersion, "1.2.3")
	}
}

func TestRootCmd_HasSearchCommand(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "search" {
			return
		}
	}
	t.Error("rootCmd does not have a 'search' subcommand")
}

func TestRootCmd_HasVersionCommand(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "version" {
			return
		}
	}
	t.Error("rootCmd does not have a 'version' subcommand")
}

func TestRootCmd_HasSetAPIKeyCommand(t *testing.T) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "set-api-key" {
			return
		}
	}
	t.Error("rootCmd does not have a 'set-api-key' subcommand")
}

func TestSearchCmd_Flags(t *testing.T) {
	flags := []struct {
		name     string
		flagType string
	}{
		{"isbn", "bool"},
		{"flat", "bool"},
		{"limit", "int"},
		{"sites", "string"},
		{"exclude", "string"},
		{"firecrawl", "bool"},
	}

	for _, f := range flags {
		t.Run(f.name, func(t *testing.T) {
			flag := searchCmd.Flags().Lookup(f.name)
			if flag == nil {
				t.Fatalf("flag --%s not found on search command", f.name)
			}
			if flag.Value.Type() != f.flagType {
				t.Errorf("flag --%s type = %q, want %q", f.name, flag.Value.Type(), f.flagType)
			}
		})
	}
}

func TestSearchCmd_RequiresArgs(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"search"})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when running search with no arguments, got nil")
	}
}

func TestVersionCommand_Output(t *testing.T) {
	origVersion := appVersion
	t.Cleanup(func() {
		appVersion = origVersion
	})

	SetVersionInfo("2.0.0")

	// The version command uses fmt.Printf which writes to os.Stdout,
	// so we capture stdout directly.
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	rootCmd.SetArgs([]string{"version"})
	execErr := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	if execErr != nil {
		t.Fatalf("version command returned error: %v", execErr)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "trbooksearch") {
		t.Errorf("version output should contain 'trbooksearch', got %q", got)
	}
	if !strings.Contains(got, "2.0.0") {
		t.Errorf("version output should contain '2.0.0', got %q", got)
	}
}

func TestRootCmd_Use(t *testing.T) {
	if rootCmd.Use != "trbooksearch" {
		t.Errorf("rootCmd.Use = %q, want %q", rootCmd.Use, "trbooksearch")
	}
}

func TestRootCmd_Short(t *testing.T) {
	if rootCmd.Short == "" {
		t.Error("rootCmd.Short should not be empty")
	}
}

func TestSearchCmd_Use(t *testing.T) {
	if !strings.Contains(searchCmd.Use, "search") {
		t.Errorf("searchCmd.Use = %q, should contain 'search'", searchCmd.Use)
	}
}

func TestSetAPIKeyCmd_Use(t *testing.T) {
	if !strings.Contains(setAPIKeyCmd.Use, "set-api-key") {
		t.Errorf("setAPIKeyCmd.Use = %q, should contain 'set-api-key'", setAPIKeyCmd.Use)
	}
}

func TestSetAPIKeyCmd_MaxArgs(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"set-api-key", "key1", "key2"})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when running set-api-key with too many arguments, got nil")
	}
}
