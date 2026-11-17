package core

const ProtocolVersion = "0.1.0"
const AgentVersion = "0.2.0"

type OSInfo struct {
	Type    string `json:"type"`
	Version string `json:"version"`
	Distro  string `json:"distro,omitempty"`
}

type EnrolledConfig struct {
	DeviceID        string `json:"device_id"`
	DeviceSecret    string `json:"device_secret"`
	ProtocolVersion string `json:"protocol_version"`
}

type Job struct {
	JobID             string `json:"job_id"`
	SessionID         string `json:"session_id"`
	DeviceID          string `json:"device_id"`
	Kind              string `json:"kind"`
	Shell             string `json:"shell"`
	Command           string `json:"command"`
	TimeoutSec        int    `json:"timeout_sec"`
	StdoutLimitBytes  int    `json:"stdout_limit_bytes"`
	StderrLimitBytes  int    `json:"stderr_limit_bytes"`
	CreatedBy         string `json:"created_by"`
}

type ServerMessage struct {
	Type string `json:"type"`
	Job  Job    `json:"job"`
}

type Heartbeat struct {
	Type         string `json:"type"`
	DeviceID     string `json:"device_id"`
	AgentVersion string `json:"agent_version"`
	UptimeSec     int64  `json:"uptime_sec"`
	ActiveJobs    int    `json:"active_jobs"`
	SentAt        string `json:"sent_at"`
}

type Hello struct {
	Type            string   `json:"type"`
	ProtocolVersion string   `json:"protocol_version"`
	AgentVersion    string   `json:"agent_version"`
	DeviceID        string   `json:"device_id"`
	OS              OSInfo   `json:"os"`
	Capabilities    []string `json:"capabilities"`
}

type JobResult struct {
	JobID           string `json:"job_id"`
	DeviceID        string `json:"device_id"`
	Status          string `json:"status"`
	ExitCode        *int   `json:"exit_code"`
	Stdout          string `json:"stdout"`
	Stderr          string `json:"stderr"`
	StdoutTruncated bool   `json:"stdout_truncated"`
	StderrTruncated bool   `json:"stderr_truncated"`
	ErrorCode       *string `json:"error_code"`
	ErrorMessage    *string `json:"error_message"`
	StartedAt       string `json:"started_at"`
	FinishedAt      string `json:"finished_at"`
	DurationMS      int64  `json:"duration_ms"`
}

type JobResultMessage struct {
	Type   string    `json:"type"`
	Result JobResult `json:"result"`
}
