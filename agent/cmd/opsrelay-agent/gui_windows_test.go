//go:build windows

package main

import (
	"testing"

	"github.com/lxn/walk"
)

func TestConnectionStatusPresentation(t *testing.T) {
	tests := []struct {
		status string
		label  string
		color  walk.Color
	}{
		{"connected", "Connected", walk.RGB(38, 165, 81)},
		{"heartbeat", "Connected", walk.RGB(38, 165, 81)},
		{"reconnecting", "Connection lost — retrying…", walk.RGB(229, 151, 30)},
		{"stopped", "Disconnected", walk.RGB(140, 150, 160)},
		{"enrollment failed", "Enrollment failed", walk.RGB(200, 58, 58)},
	}
	for _, test := range tests {
		if got := friendlyStatus(test.status); got != test.label {
			t.Errorf("friendlyStatus(%q) = %q, want %q", test.status, got, test.label)
		}
		if got := statusColor(test.status); got != test.color {
			t.Errorf("statusColor(%q) = %v, want %v", test.status, got, test.color)
		}
	}
}
