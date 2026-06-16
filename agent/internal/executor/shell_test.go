package executor

import (
	"context"
	"runtime"
	"testing"
)

func TestShellRunnerRunsShCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh is not available by default on windows")
	}
	result := NewShellRunner().Run(context.Background(), ShellRequest{
		JobID:            "job_1",
		DeviceID:         "dev_1",
		Shell:            "sh",
		Command:          "printf hello",
		TimeoutSec:       2,
		StdoutLimitBytes: 1024,
		StderrLimitBytes: 1024,
	})
	if result.Status != "completed" {
		t.Fatalf("expected completed, got %s: %s", result.Status, result.Stderr)
	}
	if result.Stdout != "hello" {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
}

func TestShellRunnerTruncatesOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh is not available by default on windows")
	}
	result := NewShellRunner().Run(context.Background(), ShellRequest{
		JobID:            "job_1",
		DeviceID:         "dev_1",
		Shell:            "sh",
		Command:          "printf 123456",
		TimeoutSec:       2,
		StdoutLimitBytes: 3,
		StderrLimitBytes: 1024,
	})
	if !result.StdoutTruncated || result.Stdout != "123" {
		t.Fatalf("expected truncated stdout, got truncated=%v stdout=%q", result.StdoutTruncated, result.Stdout)
	}
}
