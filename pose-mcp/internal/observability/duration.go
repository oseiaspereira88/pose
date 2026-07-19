package observability

import "time"

func secondsToDuration(s int) time.Duration {
	if s <= 0 {
		s = 15
	}
	return time.Duration(s) * time.Second
}
