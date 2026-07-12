package observability

import (
	"context"
	"fmt"
	"net"
)

func CheckKafkaBrokers(ctx context.Context, brokers []string) error {
	dialer := &net.Dialer{}
	var lastErr error
	for _, broker := range brokers {
		conn, err := dialer.DialContext(ctx, "tcp", broker)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("no kafka brokers configured")
}
