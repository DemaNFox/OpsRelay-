package executor

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type ShellRequest struct {
	JobID            string
	DeviceID         string
	Shell            string
	Command          string
	TimeoutSec       int
	StdoutLimitBytes int
	StderrLimitBytes int
}

type ShellResult struct {
	JobID           string
	DeviceID        string
	Status          string
	ExitCode        *int
	Stdout          string
	Stderr          string
	StdoutTruncated bool
	StderrTruncated bool
	ErrorCode       *string
	ErrorMessage    *string
	StartedAt       string
	FinishedAt      string
	DurationMS      int64
}

type ShellRunner struct{}

func NewShellRunner() *ShellRunner {
	return &ShellRunner{}
}

func (runner *ShellRunner) Run(parent context.Context, request ShellRequest) ShellResult {
	started := time.Now().UTC()
	timeout := time.Duration(request.TimeoutSec) * time.Second
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	name, args, err := commandForShell(request.Shell, request.Command)
	if err != nil {
		return resultFromError(request, started, "UNSUPPORTED_SHELL", err)
	}

	var stdout limitedBuffer
	var stderr limitedBuffer
	stdout.limit = request.StdoutLimitBytes
	stderr.limit = request.StderrLimitBytes
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	finished := time.Now().UTC()

	status := "completed"
	var exitCode *int
	var errorCode *string
	var errorMessage *string

	if ctx.Err() == context.DeadlineExceeded {
		status = "timed_out"
		code := "COMMAND_TIMED_OUT"
		message := "command timed out"
		errorCode = &code
		errorMessage = &message
	} else if err != nil {
		status = "failed"
		code := "COMMAND_FAILED"
		message := err.Error()
		errorCode = &code
		errorMessage = &message
		if exitErr, ok := err.(*exec.ExitError); ok {
			codeValue := exitErr.ExitCode()
			exitCode = &codeValue
		}
	} else {
		codeValue := 0
		exitCode = &codeValue
	}

	return ShellResult{
		JobID:           request.JobID,
		DeviceID:        request.DeviceID,
		Status:          status,
		ExitCode:        exitCode,
		Stdout:          stdout.String(),
		Stderr:          stderr.String(),
		StdoutTruncated: stdout.truncated,
		StderrTruncated: stderr.truncated,
		ErrorCode:       errorCode,
		ErrorMessage:    errorMessage,
		StartedAt:       started.Format(time.RFC3339Nano),
		FinishedAt:      finished.Format(time.RFC3339Nano),
		DurationMS:      finished.Sub(started).Milliseconds(),
	}
}

func commandForShell(shell string, command string) (string, []string, error) {
	switch shell {
	case "powershell":
		if runtime.GOOS != "windows" {
			return "", nil, errors.New("powershell is only available on windows")
		}
		return "powershell.exe", []string{"-NoProfile", "-NonInteractive", "-Command", command}, nil
	case "cmd":
		if runtime.GOOS != "windows" {
			return "", nil, errors.New("cmd is only available on windows")
		}
		return "cmd.exe", []string{"/C", command}, nil
	case "sh":
		return "sh", []string{"-c", command}, nil
	case "bash":
		return "bash", []string{"-c", command}, nil
	default:
		return "", nil, errors.New("unsupported shell")
	}
}

func resultFromError(request ShellRequest, started time.Time, code string, err error) ShellResult {
	message := err.Error()
	now := time.Now().UTC()
	return ShellResult{
		JobID:           request.JobID,
		DeviceID:        request.DeviceID,
		Status:          "failed",
		ExitCode:        nil,
		Stdout:          "",
		Stderr:          message,
		StdoutTruncated: false,
		StderrTruncated: false,
		ErrorCode:       &code,
		ErrorMessage:    &message,
		StartedAt:       started.Format(time.RFC3339Nano),
		FinishedAt:      now.Format(time.RFC3339Nano),
		DurationMS:      now.Sub(started).Milliseconds(),
	}
}

type limitedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (buffer *limitedBuffer) Write(input []byte) (int, error) {
	if buffer.limit <= 0 {
		buffer.truncated = true
		return len(input), nil
	}
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining <= 0 {
		buffer.truncated = true
		return len(input), nil
	}
	if len(input) > remaining {
		buffer.truncated = true
		_, _ = buffer.buffer.Write(input[:remaining])
		return len(input), nil
	}
	_, _ = buffer.buffer.Write(input)
	return len(input), nil
}

func (buffer *limitedBuffer) String() string {
	return strings.TrimRight(buffer.buffer.String(), "\x00")
}
