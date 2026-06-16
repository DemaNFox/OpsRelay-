//go:build !windows

package main

import (
	"fmt"
	"log/slog"
)

func runWindowsGUI(_ *slog.Logger) error {
	return fmt.Errorf("gui mode is only available on windows")
}
