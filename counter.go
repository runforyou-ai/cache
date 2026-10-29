package cache

import (
	"strconv"
)

// FormatCounter encodes n as a decimal UTF-8 counter value for [Store.Set]
// or as produced by [Counter.Incr].
func FormatCounter(n int64) []byte {
	return strconv.AppendInt(nil, n, 10)
}

// ParseCounter decodes a counter value. Empty or non-decimal input returns
// [ErrNotCounter].
func ParseCounter(b []byte) (int64, error) {
	if len(b) == 0 {
		return 0, ErrNotCounter
	}
	n, err := strconv.ParseInt(string(b), 10, 64)
	if err != nil {
		return 0, ErrNotCounter
	}
	return n, nil
}
