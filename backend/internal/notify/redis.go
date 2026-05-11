package notify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const DefaultChannel = "notify:analysis:job"

type RedisNotifier struct {
	client  *goredis.Client
	channel string
}

func NewRedisNotifier(client *goredis.Client, channel string) (*RedisNotifier, error) {
	if client == nil {
		return nil, errors.New("redis notifier client is nil")
	}
	if channel == "" {
		channel = DefaultChannel
	}

	return &RedisNotifier{
		client:  client,
		channel: channel,
	}, nil
}

func (r *RedisNotifier) NotifyJobEvent(ctx context.Context, event *JobEvent) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal job event: %w", err)
	}

	if err := r.client.Publish(ctx, r.channel, data).Err(); err != nil {
		return fmt.Errorf("publish redis job event: %w", err)
	}

	return nil
}

func (r *RedisNotifier) Subscribe(ctx context.Context, handler Handler) error {
	pubsub := r.client.Subscribe(ctx, r.channel)
	defer func() { _ = pubsub.Close() }()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return nil
			}

			var event JobEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				continue
			}
			handler(ctx, event)
		}
	}
}

func (r *RedisNotifier) Close() error {
	return r.client.Close()
}
