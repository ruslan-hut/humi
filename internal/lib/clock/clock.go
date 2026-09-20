package clock

import "time"

// Now returns the current UTC time in RFC3339.
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// Unix returns the current UTC time in seconds.
func Unix() int64 {
	return time.Now().UTC().Unix()
}
