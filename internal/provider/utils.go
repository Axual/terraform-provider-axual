package provider

import (
	"context"
	"fmt"
	"time"
)

// sleepCtx waits for d, or returns ctx.Err() at once when Terraform is cancelled (Ctrl+C).
func sleepCtx(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// RetryIf runs fn up to attempts times while retryable(err) is true, with delay between attempts.
func RetryIf(ctx context.Context, attempts int, delay time.Duration, retryable func(error) bool, fn func() error) error {
	err := fn()
	for i := 1; i < attempts && err != nil && retryable(err); i++ {
		if waitErr := sleepCtx(ctx, delay); waitErr != nil {
			return waitErr
		}
		err = fn()
	}
	return err
}

// Retry function retries the provided function `fn` for the given number of attempts, with a sleep duration between each attempt.
func Retry(attempts int, sleep time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		time.Sleep(sleep)
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
