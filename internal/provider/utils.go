package provider

import (
	"fmt"
	"time"
)

// RetryIf runs fn up to attempts times while retryable(err) is true, with delay between attempts.
func RetryIf(attempts int, delay time.Duration, retryable func(error) bool, fn func() error) error {
	err := fn()
	for i := 1; i < attempts && err != nil && retryable(err); i++ {
		time.Sleep(delay)
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
