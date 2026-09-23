package service

import (
	"sync"
	"time"
)

const (
	MessageRateLimitCount  = 10
	MessageRateLimitWindow = 5 * time.Second
)

type MessageRateLimiter struct {
	mu       sync.Mutex
	now      func() time.Time
	attempts map[string][]time.Time
}

func NewMessageRateLimiter() *MessageRateLimiter {
	return NewMessageRateLimiterWithClock(time.Now)
}

func NewMessageRateLimiterWithClock(now func() time.Time) *MessageRateLimiter {
	return &MessageRateLimiter{
		now:      now,
		attempts: make(map[string][]time.Time),
	}
}

func (l *MessageRateLimiter) Allow(userID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	cutoff := now.Add(-MessageRateLimitWindow)
	recent := l.attempts[userID][:0]
	for _, attempt := range l.attempts[userID] {
		if attempt.After(cutoff) {
			recent = append(recent, attempt)
		}
	}
	if len(recent) >= MessageRateLimitCount {
		l.attempts[userID] = recent
		return false
	}
	recent = append(recent, now)
	l.attempts[userID] = recent
	return true
}
