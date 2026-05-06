package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunPeriodic_StopsAfterCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls atomic.Int32
	done := make(chan struct{})
	go func() {
		runPeriodic(ctx, 10*time.Millisecond, func(context.Context) {
			calls.Add(1)
			if calls.Load() >= 2 {
				cancel()
			}
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runPeriodic did not stop after cancel")
	}

	if calls.Load() < 2 {
		t.Fatalf("expected at least 2 calls, got %d", calls.Load())
	}
}

func TestRunPeriodicWithInitial_ImmediateFirstCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var calls atomic.Int32
	done := make(chan struct{})

	go func() {
		runPeriodicWithInitial(ctx, time.Hour, func(context.Context) {
			if calls.Add(1) == 1 {
				cancel()
			}
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runPeriodicWithInitial should return after cancellation")
	}

	if calls.Load() != 1 {
		t.Fatalf("expected exactly 1 immediate call, got %d", calls.Load())
	}
}
