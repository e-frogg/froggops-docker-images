package main

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestLoadConfigSuccess(t *testing.T) {
	t.Setenv(envSchedule, "*/5 * * * *")
	t.Setenv(envCommand, "echo ok")
	t.Setenv(envShell, "/bin/sh")

	cfg, err := loadConfig(overrides{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Schedule != "*/5 * * * *" {
		t.Fatalf("unexpected schedule: %s", cfg.Schedule)
	}

	if cfg.Command != "echo ok" {
		t.Fatalf("unexpected command: %s", cfg.Command)
	}

	if cfg.Shell != "/bin/sh" {
		t.Fatalf("unexpected shell: %s", cfg.Shell)
	}
}

func TestLoadConfigMissing(t *testing.T) {
	t.Setenv(envSchedule, "")
	t.Setenv(envCommand, "")
	if _, err := loadConfig(overrides{}); err == nil {
		t.Fatal("expected error when env is missing")
	}
}

func TestLoadConfigOverrides(t *testing.T) {
	cfg, err := loadConfig(overrides{
		Schedule: "0 * * * *",
		Command:  "echo flag",
		Shell:    "/custom/sh",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Schedule != "0 * * * *" {
		t.Fatalf("unexpected schedule: %s", cfg.Schedule)
	}
	if cfg.Command != "echo flag" {
		t.Fatalf("unexpected command: %s", cfg.Command)
	}
	if cfg.Shell != "/custom/sh" {
		t.Fatalf("unexpected shell: %s", cfg.Shell)
	}
}

func TestBuildCommandUsesShell(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("shell detection behavior only relevant on linux build target")
	}

	originalDetector := shellDetector
	shellDetector = func(string) bool { return true }
	defer func() { shellDetector = originalDetector }()

	cmd, err := buildCommand("echo hello", "/bin/sh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(cmd.Path, "/bin/sh") {
		t.Fatalf("expected shell to be used, got %s", cmd.Path)
	}

	assertCmdArgs(t, cmd, []string{"/bin/sh", "-c", "echo hello"})
}

func TestBuildCommandWithoutShell(t *testing.T) {
	originalDetector := shellDetector
	shellDetector = func(string) bool { return false }
	defer func() { shellDetector = originalDetector }()

	cmd, err := buildCommand("echo hello world", "/bin/sh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(cmd.Path, "/bin/sh") {
		t.Fatalf("unexpected shell path: %s", cmd.Path)
	}

	assertCmdArgs(t, cmd, []string{"echo", "hello", "world"})
}

func TestParseSchedule(t *testing.T) {
	if _, err := parseSchedule("*/5 * * * *"); err != nil {
		t.Fatalf("expected valid schedule: %v", err)
	}

	if _, err := parseSchedule("invalid"); err == nil {
		t.Fatal("expected invalid cron expression")
	}
}

func TestBuildCommandEmpty(t *testing.T) {
	if _, err := buildCommand("", "/bin/sh"); err == nil {
		t.Fatal("expected error on empty command")
	}
}

func assertCmdArgs(t *testing.T, cmd *exec.Cmd, expected []string) {
	t.Helper()
	got := cmd.Args
	if len(got) != len(expected) {
		t.Fatalf("unexpected cmd args length: got %d want %d (%v)", len(got), len(expected), got)
	}
	for i := range expected {
		if os.ExpandEnv(got[i]) != os.ExpandEnv(expected[i]) {
			t.Fatalf("unexpected arg at %d: got %s want %s", i, got[i], expected[i])
		}
	}
}
