package service

import (
	"fmt"
	"sync/atomic"
	"time"
)

var sequence uint64

func nextID(prefix string) string {
	value := atomic.AddUint64(&sequence, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UTC().UnixNano(), value)
}
