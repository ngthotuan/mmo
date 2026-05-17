package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"mmo/pkg/config"
	"mmo/pkg/logger"
)

func New(cfg config.RedisConfig) (*redis.Client, error) {
	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.PingTimeout)
	defer cancel()
	if err = client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	logger.Info("connected to redis")
	return client, nil
}
