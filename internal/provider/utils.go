package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
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

func validateIfNameOrShortNamePresent(name string, shortName string, resp *datasource.ReadResponse) {
	if name == "" && shortName == "" {
		resp.Diagnostics.AddError("Missing Field", "Either `name` or `short_name` must be specified")
		return
	}
}
