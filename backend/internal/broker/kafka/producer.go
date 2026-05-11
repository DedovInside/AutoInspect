package kafka

import (
	"context"
	"fmt"

	"github.com/DedovInside/AutoInspect/backend/internal/broker"
	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
	config *config.KafkaConfig
}

func NewProducer(cfg *config.KafkaConfig) (*Producer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("kafka config is nil")
	}

	var requiredAcks kafka.RequiredAcks
	switch cfg.RequiredAcks {
	case "all", "-1":
		requiredAcks = kafka.RequireAll
	case "0":
		requiredAcks = kafka.RequireNone
	default:
		requiredAcks = kafka.RequireOne
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.Hash{},
		RequiredAcks: requiredAcks,
		MaxAttempts:  cfg.MaxRetries,
	}

	return &Producer{writer: writer, config: cfg}, nil
}

func (p *Producer) Publish(ctx context.Context, msg broker.Message) error {
	headers := make([]kafka.Header, 0, len(msg.Headers))
	for k, v := range msg.Headers {
		headers = append(headers, kafka.Header{Key: k, Value: []byte(v)})
	}

	err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic:   msg.Topic,
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: headers,
	})
	if err != nil {
		return fmt.Errorf("publish to kafka topic=%s: %w", msg.Topic, err)
	}
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
