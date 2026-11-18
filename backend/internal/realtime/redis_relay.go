package realtime

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type relayEvent struct {
	Origin string          `json:"origin"`
	Users  []string        `json:"users"`
	Data   json.RawMessage `json:"data"`
}

type RedisRelay struct {
	client          *redis.Client
	pubsub          *redis.PubSub
	hub             *Hub
	channel, origin string
	stop, done      chan struct{}
	closeOnce       sync.Once
}

func NewRedisRelay(ctx context.Context, client *redis.Client, hub *Hub, channel string) (*RedisRelay, error) {
	pubsub := client.Subscribe(ctx, channel)
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, errors.New("redis subscription failed")
	}
	r := &RedisRelay{client: client, pubsub: pubsub, hub: hub, channel: channel, origin: rand.Text(), stop: make(chan struct{}), done: make(chan struct{})}
	hub.SetPublisher(r.publish)
	go r.receive()
	return r, nil
}

func (r *RedisRelay) publish(users []string, data []byte) error {
	payload, err := json.Marshal(relayEvent{Origin: r.origin, Users: users, Data: data})
	if err != nil {
		return errors.New("invalid realtime payload")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.client.Publish(ctx, r.channel, payload).Err(); err != nil {
		return errors.New("redis publish unavailable")
	}
	return nil
}

func (r *RedisRelay) receive() {
	defer close(r.done)
	ch := r.pubsub.Channel()
	for {
		select {
		case <-r.stop:
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var event relayEvent
			if json.Unmarshal([]byte(msg.Payload), &event) != nil || event.Origin == r.origin || len(event.Data) == 0 {
				continue
			}
			for _, user := range event.Users {
				r.hub.sendLocal(user, event.Data)
			}
		}
	}
}

func (r *RedisRelay) Close() {
	r.closeOnce.Do(func() {
		r.hub.SetPublisher(nil)
		close(r.stop)
		_ = r.pubsub.Close()
		<-r.done
	})
}
