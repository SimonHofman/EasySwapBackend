package utils

import (
	"fmt"
	"time"
)

func Retry(name string, attempts int, sleep time.Duration, fn func() error) error {
	for i := 0; i < attempts; i++ {
		if err := fn(); err == nil {
			return nil
		}
		time.Sleep(sleep)
		continue
	}
	return fmt.Errorf("retry time over")
}
