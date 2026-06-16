package core

import (
	"math/rand"
	"time"
)

type Backoff struct {
	steps []time.Duration
	index int
}

func NewBackoff() *Backoff {
	return &Backoff{steps: []time.Duration{time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second, 30 * time.Second, time.Minute}}
}

func (backoff *Backoff) Next() time.Duration {
	if backoff.index >= len(backoff.steps) {
		return jitter(backoff.steps[len(backoff.steps)-1])
	}
	value := backoff.steps[backoff.index]
	backoff.index++
	return jitter(value)
}

func jitter(value time.Duration) time.Duration {
	extra := time.Duration(rand.Int63n(int64(value / 4)))
	return value + extra
}
