package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func InitRedisConnection(ctx context.Context, addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	err := client.Ping(ctx).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return client, nil
}
