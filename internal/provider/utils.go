package provider

import (
	"fmt"
	"time"
)

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
