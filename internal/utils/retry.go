package utils

import (
	"context"
	"fmt"
	"time"
)

type RetryOptions struct {
	MaxRetries int
	Backoff    time.Duration
}

func Retry[T any](ctx context.Context, fn func() (T, error), opt RetryOptions) (T, error) {
	var zero T
	maxRetries := opt.MaxRetries
	backoff := opt.Backoff
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if backoff <= 0 {
		backoff = 1 * time.Second
	}
	for i := 0; i < maxRetries; i++ {
		res, err := fn()
		if err == nil {
			return res, nil
		}
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(backoff):
			continue
		}
	}
	return zero, fmt.Errorf("failed after %d retries", maxRetries)
}

type RetryWithExponentialBackoffOptions struct {
	MaxRetries     int
	Ratio          float64
	MaxDelay       time.Duration
	InitialBackoff time.Duration
}

func RetryWithExponentialBackoff[T any](ctx context.Context, fn func() (T, error), opt RetryWithExponentialBackoffOptions) (T, error) {
	var zero T
	maxRetries := opt.MaxRetries
	ratio := opt.Ratio
	maxDelay := opt.MaxDelay
	initialBackoff := opt.InitialBackoff
	if maxRetries <= 0 {
		maxRetries = 3
	}
	if ratio <= 1 {
		ratio = 2
	}
	if initialBackoff <= 0 {
		initialBackoff = 1 * time.Second
	}
	if maxDelay <= 0 {
		maxDelay = 10 * initialBackoff
	}

	backoff := initialBackoff
	for i := 0; i < maxRetries; i++ {
		res, err := fn()
		if err == nil {
			return res, nil
		}
		backoff = time.Duration(float64(backoff) * ratio)
		if backoff > maxDelay {
			backoff = maxDelay
		}
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(backoff):
			continue
		}
	}
	return zero, fmt.Errorf("failed after %d retries", maxRetries)
}
