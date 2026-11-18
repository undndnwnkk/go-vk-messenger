package service

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/undndnwnkk/go-vk-messenger/internal/model"
)

type unavailableLimiter struct{}

func (unavailableLimiter) Check(context.Context, string) (bool, error) {
	return false, errors.New("private connection error")
}

func TestLimiterFailureDoesNotCreateMessage(t *testing.T) {
	repo := &messageRepoStub{}
	svc := NewMessageServiceWithLimiter(repo, *NewChatService(&chatRepoStub{}, UserService{}), unavailableLimiter{})
	_, err := svc.Send(context.Background(), messageUser, messageChat, model.CreateMessageRequest{Content: "hello"})
	if !errors.Is(err, ErrRateLimiterUnavailable) || repo.called {
		t.Fatalf("err=%v called=%v", err, repo.called)
	}
}

func TestRedisRateLimitSharedAcrossInstances(t *testing.T) {
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("set TEST_REDIS_URL for Redis integration tests")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	c1, c2 := redis.NewClient(opts), redis.NewClient(opts)
	defer c1.Close()
	defer c2.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	prefix := "test:" + rand.Text()
	defer c1.Del(context.Background(), prefix+":rate:alice", prefix+":rate:bob")
	limiters := []*RedisRateLimiter{NewRedisRateLimiter(c1, prefix), NewRedisRateLimiter(c2, prefix)}
	var allowed atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ok, err := limiters[i%2].Check(ctx, "alice")
			if err != nil {
				t.Error(err)
			}
			if ok {
				allowed.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if allowed.Load() != MessageRateLimitCount {
		t.Fatalf("allowed=%d", allowed.Load())
	}
	if ok, err := limiters[1].Check(ctx, "bob"); err != nil || !ok {
		t.Fatalf("other user ok=%v err=%v", ok, err)
	}
	ttl, err := c1.PTTL(ctx, prefix+":rate:alice").Result()
	if err != nil || ttl <= 0 || ttl > MessageRateLimitWindow {
		t.Fatalf("ttl=%v err=%v", ttl, err)
	}
	// Expiration frees the user's limit without any app restart.
	if err := c1.PExpire(ctx, prefix+":rate:alice", time.Millisecond).Err(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		exists, err := c1.Exists(ctx, prefix+":rate:alice").Result()
		if err != nil {
			t.Fatal(err)
		}
		if exists == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("key did not expire")
		}
		time.Sleep(time.Millisecond)
	}
	if ok, err := limiters[0].Check(ctx, "alice"); err != nil || !ok {
		t.Fatalf("after expiry ok=%v err=%v", ok, err)
	}
	c1.Close()
	if _, err := limiters[0].Check(ctx, "alice"); err == nil {
		t.Fatal("Redis failure should fail closed")
	}
}
