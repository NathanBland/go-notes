package cache

import (
	"context"
	"time"

	"github.com/valkey-io/valkey-go"
	"github.com/valkey-io/valkey-go/valkeycompat"
)

// Client exposes a tiny Valkey-backed KV interface so services can stay focused
// on behavior instead of command syntax.
type Client struct {
	raw    rawClient
	compat kvCompat
}

type rawClient interface {
	Close()
}

type stringResult interface {
	Result() (string, error)
}

type statusResult interface {
	Err() error
}

type intResult interface {
	Err() error
}

type kvCompat interface {
	Get(ctx context.Context, key string) stringResult
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) statusResult
	Del(ctx context.Context, keys ...string) intResult
}

type compatAdapter struct {
	raw valkeycompat.Cmdable
}

func (c compatAdapter) Get(ctx context.Context, key string) stringResult {
	return c.raw.Get(ctx, key)
}

func (c compatAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) statusResult {
	return c.raw.Set(ctx, key, value, expiration)
}

func (c compatAdapter) Del(ctx context.Context, keys ...string) intResult {
	return c.raw.Del(ctx, keys...)
}

// New opens a Valkey client for the configured local or container address.
func New(address, password string) (*Client, error) {
	options := valkey.ClientOption{InitAddress: []string{address}}
	if password != "" {
		options.Password = password
	}
	client, err := valkey.NewClient(options)
	if err != nil {
		return nil, err
	}
	return &Client{
		raw:    client,
		compat: compatAdapter{raw: valkeycompat.NewAdapter(client)},
	}, nil
}

// Get returns the cached value plus a hit flag.
func (c *Client) Get(ctx context.Context, key string) (string, bool) {
	value, err := c.compat.Get(ctx, key).Result()
	if err != nil {
		return "", false
	}
	return value, true
}

// Set stores a value with the provided TTL.
func (c *Client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.compat.Set(ctx, key, value, ttl).Err()
}

// Delete removes one or more keys.
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.compat.Del(ctx, keys...).Err()
}

// Close shuts down the underlying Valkey client connection pool.
func (c *Client) Close() error {
	if c.raw != nil {
		c.raw.Close()
	}
	return nil
}
