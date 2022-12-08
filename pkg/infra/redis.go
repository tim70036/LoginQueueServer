package infra

import (
	"context"

	"github.com/go-redis/redis/v8"
)

var (
	RedisClient *redis.Client
)

func init() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:8787",
		Password: "", // no password set
		DB:       0,  // use default DB
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			BaseLogger.Info("redis connected")
			return nil
		},
	})

}
