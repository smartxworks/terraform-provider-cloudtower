package utils

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testContext struct {
	retries    *int
	cancelFunc context.CancelFunc
}

func TestRetry(t *testing.T) {
	tests := []struct {
		name       string
		fn         func(*testContext) func() (string, error)
		maxRetries int
		backoff    time.Duration
		wantErr    bool
		wantRes    string
	}{
		{
			name: "success on first try",
			fn: func(_ *testContext) func() (string, error) {
				return func() (string, error) {
					return "success", nil
				}
			},
			maxRetries: 3,
			backoff:    time.Millisecond * 100,
			wantErr:    false,
			wantRes:    "success",
		},
		{
			name: "success after retries",
			fn: func(ctx *testContext) func() (string, error) {
				return func() (string, error) {
					if *ctx.retries == 0 {
						*ctx.retries++
						return "", errors.New("temporary error")
					}
					return "success", nil
				}
			},
			maxRetries: 3,
			backoff:    time.Millisecond * 100,
			wantErr:    false,
			wantRes:    "success",
		},
		{
			name: "failed on all retries",
			fn: func(_ *testContext) func() (string, error) {
				return func() (string, error) {
					return "", errors.New("temporary error")
				}
			},
			maxRetries: 3,
			backoff:    time.Millisecond * 100,
			wantErr:    true,
			wantRes:    "",
		},
		{
			name: "context cancellation",
			fn: func(ctx *testContext) func() (string, error) {
				return func() (string, error) {
					if *ctx.retries == 0 {
						*ctx.retries++
						ctx.cancelFunc()
						return "", errors.New("temporary error")
					}
					return "success", nil
				}
			},
			maxRetries: 3,
			backoff:    time.Second,
			wantErr:    true,
			wantRes:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			retries := 0
			testCtx := &testContext{
				retries:    &retries,
				cancelFunc: func() {},
			}
			if tt.name == "context cancellation" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
				testCtx.cancelFunc = cancel
			}

			res, err := Retry(ctx, tt.fn(testCtx), RetryOptions{
				MaxRetries: tt.maxRetries,
				Backoff:    tt.backoff,
			})
			if tt.wantErr {
				if err == nil {
					t.Errorf("Retry() error = %v, want %v", err, tt.wantErr)
				}
			} else {
				if err != nil {
					t.Errorf("Retry() error = %v, want nil", err)
				} else if res != tt.wantRes {
					t.Errorf("Retry() res = %v, want %v", res, tt.wantRes)
				}
			}
		})
	}
}

func TestRetryWithExponentialBackoff(t *testing.T) {
	tests := []struct {
		name           string
		fn             func(*testContext) func() (string, error)
		maxRetries     int
		ratio          float64
		initialBackoff time.Duration
		maxDelay       time.Duration
		wantErr        bool
		wantRes        string
	}{
		{
			name: "success on first try",
			fn: func(_ *testContext) func() (string, error) {
				return func() (string, error) {
					return "success", nil
				}
			},
			maxRetries:     3,
			ratio:          2.0,
			initialBackoff: time.Millisecond * 100,
			maxDelay:       time.Second,
			wantErr:        false,
			wantRes:        "success",
		},
		{
			name: "success after retries",
			fn: func(ctx *testContext) func() (string, error) {
				return func() (string, error) {
					if *ctx.retries == 0 {
						*ctx.retries++
						return "", errors.New("temporary error")
					}
					return "success", nil
				}
			},
			maxRetries:     3,
			ratio:          2.0,
			initialBackoff: time.Millisecond * 100,
			maxDelay:       time.Second,
			wantErr:        false,
			wantRes:        "success",
		},
		{
			name: "failed on all retries",
			fn: func(_ *testContext) func() (string, error) {
				return func() (string, error) {
					return "", errors.New("reached max retries")
				}
			},
			maxRetries:     3,
			ratio:          2.0,
			initialBackoff: time.Millisecond * 100,
			maxDelay:       time.Second,
			wantErr:        true,
			wantRes:        "",
		},
		{
			name: "context cancellation",
			fn: func(ctx *testContext) func() (string, error) {
				return func() (string, error) {
					if *ctx.retries == 0 {
						*ctx.retries++
						ctx.cancelFunc()
						return "", errors.New("temporary error")
					}
					return "success", nil
				}
			},
			maxRetries:     3,
			ratio:          2.0,
			initialBackoff: time.Millisecond * 100,
			maxDelay:       time.Second,
			wantErr:        true,
			wantRes:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			retries := 0
			testCtx := &testContext{
				retries:    &retries,
				cancelFunc: func() {},
			}
			if tt.name == "context cancellation" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
				testCtx.cancelFunc = cancel
			}

			res, err := RetryWithExponentialBackoff(ctx, tt.fn(testCtx), RetryWithExponentialBackoffOptions{
				MaxRetries:     tt.maxRetries,
				Ratio:          tt.ratio,
				MaxDelay:       tt.maxDelay,
				InitialBackoff: tt.initialBackoff,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("RetryWithExponentialBackoff() error = %v, wantErr %v", err, tt.wantErr)
			} else if res != tt.wantRes {
				t.Errorf("RetryWithExponentialBackoff() res = %v, want %v", res, tt.wantRes)
			}
		})
	}
}
