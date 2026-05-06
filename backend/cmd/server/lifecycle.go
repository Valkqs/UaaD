package main

import (
	"context"
	"time"
)

func runPeriodic(ctx context.Context, interval time.Duration, job func(context.Context)) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			job(ctx)
		}
	}
}

func runPeriodicWithInitial(ctx context.Context, interval time.Duration, job func(context.Context)) {
	job(ctx)
	runPeriodic(ctx, interval, job)
}
