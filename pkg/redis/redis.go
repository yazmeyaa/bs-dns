package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/yazmeyaa/bs-dns/internal/config"
)

func InitRedisConnection(ctx context.Context, config *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Host,
		Username: config.Redis.Username,
		Password: config.Redis.Password,
		DB:       config.Redis.Database,
	})

	err := client.Ping(ctx).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return client, nil
}
