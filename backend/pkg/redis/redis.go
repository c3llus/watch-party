package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"watch-party/pkg/config"
	"watch-party/pkg/logger"

	"github.com/redis/go-redis/v9"
)

// Client wraps redis client with additional functionality
type Client struct {
	client *redis.Client
}

// NewClient creates a new Redis client
func NewClient(cfg *config.Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := rdb.Ping(ctx)
	if result.Err() != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", result.Err())
	}

	logger.Info("Connected to Redis successfully")

	return &Client{
		client: rdb,
	}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}

// Publish publishes a message to a Redis channel
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	result := c.client.Publish(ctx, channel, data)
	if result.Err() != nil {
		return fmt.Errorf("failed to publish message: %w", result.Err())
	}

	return nil
}

// Subscribe subscribes to a Redis channel
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.client.Subscribe(ctx, channels...)
}

// Set sets a key-value pair with expiration
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	result := c.client.Set(ctx, key, data, expiration)
	if result.Err() != nil {
		return fmt.Errorf("failed to set key: %w", result.Err())
	}

	return nil
}

// Get gets a value by key
func (c *Client) Get(ctx context.Context, key string, dest interface{}) error {
	result := c.client.Get(ctx, key)
	if result.Err() != nil {
		if result.Err() == redis.Nil {
			return fmt.Errorf("key not found: %s", key)
		}
		return fmt.Errorf("failed to get key: %w", result.Err())
	}

	data, err := result.Bytes()
	if err != nil {
		return fmt.Errorf("failed to get bytes: %w", err)
	}

	err = json.Unmarshal(data, dest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

// Delete deletes a key
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	result := c.client.Del(ctx, keys...)
	if result.Err() != nil {
		return fmt.Errorf("failed to delete keys: %w", result.Err())
	}

	return nil
}

// SetAdd adds members to a set
func (c *Client) SetAdd(ctx context.Context, key string, members ...interface{}) error {
	result := c.client.SAdd(ctx, key, members...)
	if result.Err() != nil {
		return fmt.Errorf("failed to add to set: %w", result.Err())
	}

	return nil
}

// SetRemove removes members from a set
func (c *Client) SetRemove(ctx context.Context, key string, members ...interface{}) error {
	result := c.client.SRem(ctx, key, members...)
	if result.Err() != nil {
		return fmt.Errorf("failed to remove from set: %w", result.Err())
	}

	return nil
}

// SetMembers gets all members of a set
func (c *Client) SetMembers(ctx context.Context, key string) ([]string, error) {
	result := c.client.SMembers(ctx, key)
	if result.Err() != nil {
		return nil, fmt.Errorf("failed to get set members: %w", result.Err())
	}

	return result.Val(), nil
}

// SetIsMember checks if a value is member of a set
func (c *Client) SetIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	result := c.client.SIsMember(ctx, key, member)
	if result.Err() != nil {
		return false, fmt.Errorf("failed to check set membership: %w", result.Err())
	}

	return result.Val(), nil
}

// Pipeline returns a Redis pipeline for batch operations
func (c *Client) Pipeline() redis.Pipeliner {
	return c.client.Pipeline()
}

// HSet sets field-value pairs in a hash
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	result := c.client.HSet(ctx, key, values...)
	if result.Err() != nil {
		return fmt.Errorf("failed to set hash fields: %w", result.Err())
	}
	return nil
}

// HGet gets a field value from a hash
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	result := c.client.HGet(ctx, key, field)
	if result.Err() != nil {
		if result.Err() == redis.Nil {
			return "", fmt.Errorf("field not found")
		}
		return "", fmt.Errorf("failed to get hash field: %w", result.Err())
	}
	return result.Val(), nil
}

// HGetAll gets all field-value pairs from a hash
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	result := c.client.HGetAll(ctx, key)
	if result.Err() != nil {
		return nil, fmt.Errorf("failed to get all hash fields: %w", result.Err())
	}
	return result.Val(), nil
}

// HDel deletes fields from a hash
func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	result := c.client.HDel(ctx, key, fields...)
	if result.Err() != nil {
		return fmt.Errorf("failed to delete hash fields: %w", result.Err())
	}
	return nil
}

// ZAdd adds members to a sorted set
func (c *Client) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	result := c.client.ZAdd(ctx, key, members...)
	if result.Err() != nil {
		return fmt.Errorf("failed to add to sorted set: %w", result.Err())
	}
	return nil
}

// ZRevRange gets members from a sorted set in reverse order
func (c *Client) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	result := c.client.ZRevRange(ctx, key, start, stop)
	if result.Err() != nil {
		return nil, fmt.Errorf("failed to get sorted set range: %w", result.Err())
	}
	return result.Val(), nil
}

// ZRangeByScore gets members from a sorted set by score range
func (c *Client) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	result := c.client.ZRangeByScore(ctx, key, opt)
	if result.Err() != nil {
		return nil, fmt.Errorf("failed to get sorted set by score: %w", result.Err())
	}
	return result.Val(), nil
}

// ZRem removes members from a sorted set
func (c *Client) ZRem(ctx context.Context, key string, members ...interface{}) error {
	result := c.client.ZRem(ctx, key, members...)
	if result.Err() != nil {
		return fmt.Errorf("failed to remove from sorted set: %w", result.Err())
	}
	return nil
}

// SetNX sets a key only if it doesn't exist
func (c *Client) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	result := c.client.SetNX(ctx, key, value, expiration)
	if result.Err() != nil {
		return false, fmt.Errorf("failed to set key if not exists: %w", result.Err())
	}
	return result.Val(), nil
}

// Expire sets expiration for a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	result := c.client.Expire(ctx, key, expiration)
	if result.Err() != nil {
		return fmt.Errorf("failed to set expiration: %w", result.Err())
	}
	return nil
}
