package kafka

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/broker"
	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	config *config.KafkaConfig
	writer *kafka.Writer
}

func NewConsumer(cfg *config.KafkaConfig) (*Consumer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("kafka config is nil")
	}

	writer := &kafka.Writer{
		Addr:        kafka.TCP(cfg.Brokers...),
		Balancer:    &kafka.Hash{},
		MaxAttempts: cfg.MaxRetries,
	}

	return &Consumer{config: cfg, writer: writer}, nil
}

func (c *Consumer) Subscribe(ctx context.Context, topic string, handler broker.MessageHandler) error {
	reader := c.newReader(topic)
	defer func() {
		_ = reader.Close()
	}()

	return c.consumeLoop(ctx, reader, handler)
}

func (c *Consumer) newReader(topic string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  c.config.Brokers,
		GroupID:  c.config.ConsumerGroupID,
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
}

func (c *Consumer) consumeLoop(ctx context.Context, reader *kafka.Reader, handler broker.MessageHandler) error {
	fetchBackoff := c.initialFetchBackoff()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			done, next := c.handleFetchError(ctx, err, fetchBackoff)
			if done {
				return nil
			}
			fetchBackoff = next
			continue
		}

		fetchBackoff = c.initialFetchBackoff()

		if err := c.processMessage(ctx, reader, &msg, handler); err != nil {
			return err
		}
	}
}

func (c *Consumer) initialFetchBackoff() time.Duration {
	fetchBackoff := c.config.ConsumerFetchBackoffMin
	if fetchBackoff <= 0 {
		return 200 * time.Millisecond
	}
	return fetchBackoff
}

func (c *Consumer) handleFetchError(ctx context.Context, _ error, currentBackoff time.Duration) (bool, time.Duration) {
	if ctx.Err() != nil {
		return true, currentBackoff
	}

	sleepWithContext(ctx, currentBackoff)
	return false, nextBackoff(currentBackoff, c.config.ConsumerFetchBackoffMax)
}

func (c *Consumer) processMessage(ctx context.Context, reader *kafka.Reader, msg *kafka.Message, handler broker.MessageHandler) error {
	err := handler(ctx, broker.Message{
		Topic:   msg.Topic,
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: headersToMap(msg.Headers),
	})

	if err != nil {
		if handleErr := c.handleProcessingError(ctx, reader, msg, err); handleErr != nil {
			if ctx.Err() != nil {
				return nil
			}
			return handleErr
		}
		return nil
	}

	if err := reader.CommitMessages(ctx, *msg); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("commit processed message: %w", err)
	}

	return nil
}

func headersToMap(headers []kafka.Header) map[string]string {
	out := make(map[string]string, len(headers))
	for _, h := range headers {
		out[h.Key] = string(h.Value)
	}
	return out
}

func (c *Consumer) handleProcessingError(ctx context.Context, reader *kafka.Reader, msg *kafka.Message, processErr error) error {
	attempt := parseAttempt(msg.Headers)
	nextAttempt := attempt + 1

	if nextAttempt <= c.maxRetryAttempts() {
		retryMsg := *msg
		retryMsg.Headers = upsertHeader(msg.Headers, "x-retry-attempt", strconv.Itoa(nextAttempt))
		retryMsg.Headers = upsertHeader(retryMsg.Headers, "x-last-error", processErr.Error())

		if err := c.writer.WriteMessages(ctx, retryMsg); err != nil {
			return fmt.Errorf("publish retry message: %w", err)
		}

		if err := reader.CommitMessages(ctx, *msg); err != nil {
			return fmt.Errorf("commit retried message: %w", err)
		}

		sleepWithContext(ctx, c.retryBackoff(nextAttempt))
		return nil
	}

	dlqTopic := c.config.TopicDLQ
	if dlqTopic == "" {
		dlqTopic = msg.Topic + ".dlq"
	}

	dlqMsg := kafka.Message{
		Topic: dlqTopic,
		Key:   msg.Key,
		Value: msg.Value,
		Headers: upsertHeader(
			upsertHeader(msg.Headers, "x-last-error", processErr.Error()),
			"x-original-topic",
			msg.Topic,
		),
	}

	if err := c.writer.WriteMessages(ctx, dlqMsg); err != nil {
		return fmt.Errorf("publish dlq message: %w", err)
	}

	if err := reader.CommitMessages(ctx, *msg); err != nil {
		return fmt.Errorf("commit dlq-routed message: %w", err)
	}

	return nil
}

func parseAttempt(headers []kafka.Header) int {
	for _, h := range headers {
		if h.Key != "x-retry-attempt" {
			continue
		}
		n, err := strconv.Atoi(string(h.Value))
		if err != nil || n < 0 {
			return 0
		}
		return n
	}
	return 0
}

func upsertHeader(headers []kafka.Header, key, value string) []kafka.Header {
	updated := make([]kafka.Header, 0, len(headers)+1)
	replaced := false

	for _, h := range headers {
		if h.Key == key {
			h.Value = []byte(value)
			replaced = true
		}
		updated = append(updated, h)
	}

	if !replaced {
		updated = append(updated, kafka.Header{Key: key, Value: []byte(value)})
	}

	return updated
}

func (c *Consumer) maxRetryAttempts() int {
	if c.config.ConsumerMaxRetries <= 0 {
		return 3
	}
	return c.config.ConsumerMaxRetries
}

func (c *Consumer) retryBackoff(attempt int) time.Duration {
	minBackoff := c.config.ConsumerRetryBackoffMin
	maxBackoff := c.config.ConsumerRetryBackoffMax
	if minBackoff <= 0 {
		minBackoff = 200 * time.Millisecond
	}
	if maxBackoff <= 0 {
		maxBackoff = 5 * time.Second
	}

	backoff := minBackoff
	for i := 1; i < attempt; i++ {
		backoff *= 2
		if backoff >= maxBackoff {
			return maxBackoff
		}
	}
	return backoff
}

func nextBackoff(current, maxBackoff time.Duration) time.Duration {
	if current <= 0 {
		current = 200 * time.Millisecond
	}
	if maxBackoff <= 0 {
		maxBackoff = 5 * time.Second
	}
	next := current * 2
	if next > maxBackoff {
		return maxBackoff
	}
	return next
}

func sleepWithContext(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func (c *Consumer) Close() error {
	return c.writer.Close()
}
