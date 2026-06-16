package core

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/opsrelay/agent/internal/config"
	"github.com/opsrelay/agent/internal/executor"
)

type ShellRunner interface {
	Run(ctx context.Context, request executor.ShellRequest) executor.ShellResult
}

type Rebooter interface {
	Reboot(ctx context.Context, reason string) error
}

type Connector interface {
	Connect(ctx context.Context, deviceID string, deviceSecret string) (*websocket.Conn, *http.Response, error)
}

type Agent struct {
	config   config.Config
	client   Connector
	shell    ShellRunner
	rebooter Rebooter
	logger   *slog.Logger
	started  time.Time
	osInfo   OSInfo
	caps     []string
	status   func(string)
}

func NewAgent(cfg config.Config, client Connector, shell ShellRunner, rebooter Rebooter, osInfo OSInfo, capabilities []string, logger *slog.Logger) *Agent {
	return &Agent{
		config:   cfg,
		client:   client,
		shell:    shell,
		rebooter: rebooter,
		logger:   logger,
		started:  time.Now(),
		osInfo:   osInfo,
		caps:     capabilities,
	}
}

func (agent *Agent) SetStatusHandler(handler func(string)) {
	agent.status = handler
}

func (agent *Agent) Run(ctx context.Context) error {
	backoff := NewBackoff()
	for {
		if err := ctx.Err(); err != nil {
			agent.setStatus("stopped")
			return err
		}
		agent.setStatus("connecting")
		err := agent.runOnce(ctx)
		if err != nil {
			agent.setStatus("reconnecting")
			agent.logger.Warn("agent connection ended", slog.String("error", err.Error()))
		}
		delay := backoff.Next()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

func (agent *Agent) runOnce(ctx context.Context) error {
	conn, _, err := agent.client.Connect(ctx, agent.config.DeviceID, agent.config.DeviceSecret)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.WriteJSON(Hello{
		Type:            "hello",
		ProtocolVersion: ProtocolVersion,
		AgentVersion:    AgentVersion,
		DeviceID:        agent.config.DeviceID,
		OS:              agent.osInfo,
		Capabilities:    agent.caps,
	}); err != nil {
		return err
	}
	agent.setStatus("connected")

	done := make(chan error, 1)
	go agent.readLoop(ctx, conn, done)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			return err
		case <-ticker.C:
			heartbeat := Heartbeat{
				Type:         "heartbeat",
				DeviceID:     agent.config.DeviceID,
				AgentVersion: AgentVersion,
				UptimeSec:    int64(time.Since(agent.started).Seconds()),
				ActiveJobs:   0,
				SentAt:       time.Now().UTC().Format(time.RFC3339Nano),
			}
			if err := conn.WriteJSON(heartbeat); err != nil {
				return err
			}
		}
	}
}

func (agent *Agent) setStatus(status string) {
	if agent.status != nil {
		agent.status(status)
	}
}

func (agent *Agent) readLoop(ctx context.Context, conn *websocket.Conn, done chan<- error) {
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			done <- err
			return
		}
		var message ServerMessage
		if err := json.Unmarshal(raw, &message); err != nil {
			agent.logger.Warn("invalid server message", slog.String("error", err.Error()))
			continue
		}
		if message.Type != "job" {
			continue
		}
		go agent.handleJob(ctx, conn, message.Job)
	}
}

func (agent *Agent) handleJob(ctx context.Context, conn *websocket.Conn, job Job) {
	agent.logger.Info("job received", slog.String("job_id", job.JobID), slog.String("kind", job.Kind))
	var result JobResult
	if job.Kind == "shell.exec" {
		result = agent.runShellJob(ctx, job)
	} else if job.Kind == "system.reboot" {
		result = agent.runRebootJob(ctx, job)
	} else {
		result = failedResult(job, "UNSUPPORTED_JOB_KIND", "Unsupported job kind")
	}
	if err := conn.WriteJSON(JobResultMessage{Type: "job_result", Result: result}); err != nil {
		agent.logger.Error("failed to send job result", slog.String("job_id", job.JobID), slog.String("error", err.Error()))
	}
}

func (agent *Agent) runShellJob(ctx context.Context, job Job) JobResult {
	result := agent.shell.Run(ctx, executor.ShellRequest{
		JobID:            job.JobID,
		DeviceID:         job.DeviceID,
		Shell:            job.Shell,
		Command:          job.Command,
		TimeoutSec:       job.TimeoutSec,
		StdoutLimitBytes: job.StdoutLimitBytes,
		StderrLimitBytes: job.StderrLimitBytes,
	})
	return JobResult{
		JobID:           result.JobID,
		DeviceID:        result.DeviceID,
		Status:          result.Status,
		ExitCode:        result.ExitCode,
		Stdout:          result.Stdout,
		Stderr:          result.Stderr,
		StdoutTruncated: result.StdoutTruncated,
		StderrTruncated: result.StderrTruncated,
		ErrorCode:       result.ErrorCode,
		ErrorMessage:    result.ErrorMessage,
		StartedAt:       result.StartedAt,
		FinishedAt:      result.FinishedAt,
		DurationMS:      result.DurationMS,
	}
}

func (agent *Agent) runRebootJob(ctx context.Context, job Job) JobResult {
	started := time.Now().UTC()
	err := agent.rebooter.Reboot(ctx, job.Command)
	finished := time.Now().UTC()
	if err != nil {
		code := "REBOOT_FAILED"
		message := err.Error()
		return JobResult{
			JobID:           job.JobID,
			DeviceID:        job.DeviceID,
			Status:          "failed",
			ExitCode:        nil,
			Stdout:          "",
			Stderr:          message,
			StdoutTruncated: false,
			StderrTruncated: false,
			ErrorCode:       &code,
			ErrorMessage:    &message,
			StartedAt:       started.Format(time.RFC3339Nano),
			FinishedAt:      finished.Format(time.RFC3339Nano),
			DurationMS:      finished.Sub(started).Milliseconds(),
		}
	}
	exitCode := 0
	return JobResult{
		JobID:           job.JobID,
		DeviceID:        job.DeviceID,
		Status:          "completed",
		ExitCode:        &exitCode,
		Stdout:          "reboot requested\n",
		Stderr:          "",
		StdoutTruncated: false,
		StderrTruncated: false,
		ErrorCode:       nil,
		ErrorMessage:    nil,
		StartedAt:       started.Format(time.RFC3339Nano),
		FinishedAt:      finished.Format(time.RFC3339Nano),
		DurationMS:      finished.Sub(started).Milliseconds(),
	}
}

func failedResult(job Job, code string, message string) JobResult {
	started := time.Now().UTC()
	return JobResult{
		JobID:           job.JobID,
		DeviceID:        job.DeviceID,
		Status:          "failed",
		ExitCode:        nil,
		Stdout:          "",
		Stderr:          message,
		StdoutTruncated: false,
		StderrTruncated: false,
		ErrorCode:       &code,
		ErrorMessage:    &message,
		StartedAt:       started.Format(time.RFC3339Nano),
		FinishedAt:      time.Now().UTC().Format(time.RFC3339Nano),
		DurationMS:      0,
	}
}
