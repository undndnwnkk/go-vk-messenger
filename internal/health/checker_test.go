package health

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestCheckerDependenciesAndDeadline(t *testing.T) {
	failed := errors.New("dependency down")
	calls := 0
	c := Checker{Checks: []func(context.Context) error{
		func(context.Context) error { calls++; return nil },
		func(context.Context) error { calls++; return failed },
	}}
	if err := c.Ping(context.Background()); !errors.Is(err, failed) || calls != 2 {
		t.Fatalf("Ping = %v, calls = %d", err, calls)
	}
	c.Checks = []func(context.Context) error{func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := c.Ping(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Ping = %v", err)
	}
}

func TestCheckerReportsUnavailableRedis(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	_ = client.Close()
	c := Checker{Checks: []func(context.Context) error{
		func(context.Context) error { return nil },
		func(ctx context.Context) error { return client.Ping(ctx).Err() },
	}}
	if err := c.Ping(context.Background()); err == nil {
		t.Fatal("readiness must fail when configured Redis is unavailable")
	}
}
