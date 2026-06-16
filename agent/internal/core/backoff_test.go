package core

import (
	"testing"
	"time"
)

func TestBackoffDoesNotExceedExpectedRange(t *testing.T) {
	backoff := NewBackoff()
	first := backoff.Next()
	if first < time.Second || first > 1250*time.Millisecond {
		t.Fatalf("unexpected first backoff: %s", first)
	}
}
