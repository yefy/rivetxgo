package session

import (
	"sync/atomic"
	"time"
)

var sessionId atomic.Uint64

func init() {
	sessionId.Store(uint64(time.Now().UnixMilli()))
}

func SessionId() uint64 {
	return sessionId.Add(1)
}
