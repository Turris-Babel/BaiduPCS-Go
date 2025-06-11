package retry

import "time"

// Backoff returns the wait duration before the next retry attempt.
func Backoff(retry int) time.Duration {
	if retry < 3 {
		return 2 * time.Duration(retry) * time.Second
	}
	return 6 * time.Second
}
