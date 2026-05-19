//go:build integration

package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/broker"
	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/DedovInside/AutoInspect/backend/internal/testsupport"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
)

func TestKafkaProducerConsumerPublishConsumeMessage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	brokers := kafkaBrokers(t, ctx)
	topic := "autoinspect.integration.analysis.result"
	createKafkaTopic(t, ctx, brokers[0], topic)
	cfg := &config.KafkaConfig{
		Brokers:                 brokers,
		TopicAnalysisResult:     topic,
		RequiredAcks:            "all",
		MaxRetries:              3,
		ConsumerGroupID:         "autoinspect-integration-test",
		ConsumerMaxRetries:      1,
		ConsumerFetchBackoffMin: 50 * time.Millisecond,
		ConsumerFetchBackoffMax: 200 * time.Millisecond,
		ConsumerRetryBackoffMin: 50 * time.Millisecond,
		ConsumerRetryBackoffMax: 200 * time.Millisecond,
		TopicDLQ:                "autoinspect.integration.dlq",
		SecurityProtocol:        "PLAINTEXT",
	}

	producer, err := NewProducer(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, producer.Close()) })

	consumer, err := NewConsumer(cfg)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, consumer.Close()) })

	received := make(chan broker.Message, 1)
	consumeCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	go func() {
		_ = consumer.Subscribe(consumeCtx, topic, func(_ context.Context, msg broker.Message) error {
			received <- msg
			cancel()
			return nil
		})
	}()
	time.Sleep(500 * time.Millisecond)

	message := broker.Message{
		Topic: topic,
		Key:   []byte("analysis-1"),
		Value: []byte("protobuf-payload"),
		Headers: map[string]string{
			"content-type": "application/protobuf",
		},
	}
	require.NoError(t, producer.Publish(ctx, message))

	select {
	case got := <-received:
		require.Equal(t, topic, got.Topic)
		require.Equal(t, message.Key, got.Key)
		require.Equal(t, message.Value, got.Value)
		require.Equal(t, "application/protobuf", got.Headers["content-type"])
	case <-time.After(15 * time.Second):
		t.Fatal("timed out waiting for kafka message")
	}
}

func kafkaBrokers(t *testing.T, ctx context.Context) []string {
	t.Helper()

	container, err := tckafka.Run(ctx, "confluentinc/confluent-local:7.5.0", tckafka.WithClusterID("autoinspect-test-cluster"))
	if err != nil {
		testsupport.SkipIfDockerUnavailable(t, err)
		require.NoError(t, err)
	}
	testcontainers.CleanupContainer(t, container)

	brokers, err := container.Brokers(ctx)
	require.NoError(t, err)
	return brokers
}

func createKafkaTopic(t *testing.T, ctx context.Context, brokerAddress, topic string) {
	t.Helper()

	conn, err := kafkago.DialContext(ctx, "tcp", brokerAddress)
	require.NoError(t, err)
	defer func() { require.NoError(t, conn.Close()) }()

	err = conn.CreateTopics(kafkago.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	require.NoError(t, err)
}
