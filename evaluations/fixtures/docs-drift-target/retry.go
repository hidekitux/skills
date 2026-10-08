// Package retry runs an operation until it succeeds.
package retry

import "time"

// Do calls fn up to attempts times and sleeps delay between failed calls. It
// returns nil after the first success, or the last error.
func Do(attempts int, delay time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		if i < attempts-1 {
			time.Sleep(delay)
		}
	}
	return err
}
