package service

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRateLimiter struct {
	client *redis.Client
	prefix string
}

func NewRedisRateLimiter(client *redis.Client, prefix string) *RedisRateLimiter {
	return &RedisRateLimiter{client: client, prefix: prefix}
}

func (l *RedisRateLimiter) Check(ctx context.Context, userID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	allowed, err := slidingWindowScript.Run(ctx, l.client, []string{l.prefix + ":rate:" + userID}, MessageRateLimitWindow.Milliseconds(), MessageRateLimitCount, rand.Text()).Int()
	return allowed == 1, err
}

// Use Redis time, a unique member and a single atomic script across all replicas.
// Rejected attempts neither consume an extra slot nor extend the window.
var slidingWindowScript = redis.NewScript(`
local t = redis.call('TIME')
local now = t[1] * 1000 + math.floor(t[2] / 1000)
local window = tonumber(ARGV[1])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now - window)
if redis.call('ZCARD', KEYS[1]) >= tonumber(ARGV[2]) then return 0 end
redis.call('ZADD', KEYS[1], now, ARGV[3])
redis.call('PEXPIRE', KEYS[1], window)
return 1
`)
