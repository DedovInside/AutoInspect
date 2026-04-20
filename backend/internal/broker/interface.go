package broker

import "context"

type Message struct {
	Topic   string
	Key     []byte
	Value   []byte
	Headers map[string]string
}

type Publisher interface {
	Publish(ctx context.Context, msg Message) error
	Close() error
}

type MessageHandler func(ctx context.Context, msg Message) error

type Subscriber interface {
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error
	Close() error
}

type Broker interface {
	Publisher
	Subscriber
}
