package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/config"
	goredis "github.com/redis/go-redis/v9"
)

const (
	denylistPrefix   = "auth:denylist:access:"
	oauthStatePrefix = "auth:oauth:state:"
)

type Client struct {
	client *goredis.Client
}

func New(cfg config.RedisConfig) *Client {
	return &Client{
		client: goredis.NewClient(&goredis.Options{
			Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
			Password:     cfg.Password,
			DB:           cfg.DB,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		}),
	}
}

func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) SetDenylistJTI(ctx context.Context, jti string, ttl time.Duration) error {
	return c.client.Set(ctx, denylistPrefix+jti, "1", ttl).Err()
}

func (c *Client) IsDenylistedJTI(ctx context.Context, jti string) (bool, error) {
	n, err := c.client.Exists(ctx, denylistPrefix+jti).Result()

	if err != nil {
		return false, err
	}

	return n > 0, nil
}

func (c *Client) StoreOAuthState(ctx context.Context, state string, ttl time.Duration) error {
	return c.client.Set(ctx, oauthStatePrefix+state, "1", ttl).Err()
}

func (c *Client) ConsumeOAuthState(ctx context.Context, state string) (bool, error) {
	result, err := c.client.Del(ctx, oauthStatePrefix+state).Result()

	if err != nil {
		return false, err
	}

	return result > 0, nil
}
