package platform

import (
	"context"
	"errors"
	"os/exec"
	"runtime"

	"github.com/opsrelay/agent/internal/core"
)

func OSInfo() core.OSInfo {
	return core.OSInfo{Type: osType(), Version: runtime.GOOS}
}

func Capabilities() []string {
	if runtime.GOOS == "windows" {
		return []string{"shell.powershell", "shell.cmd", "reboot"}
	}
	return []string{"shell.sh", "shell.bash", "reboot"}
}

func osType() string {
	if runtime.GOOS == "windows" {
		return "windows"
	}
	return "linux"
}

type Rebooter struct {
	allow bool
}

func NewRebooter(allow bool) *Rebooter {
	return &Rebooter{allow: allow}
}

func (rebooter *Rebooter) Reboot(ctx context.Context, reason string) error {
	if !rebooter.allow {
		return errors.New("reboot disabled in agent config; set allow_reboot=true after operator approval")
	}
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "shutdown.exe", "/r", "/t", "0", "/c", reason).Run()
	}
	return exec.CommandContext(ctx, "shutdown", "-r", "now", reason).Run()
}
