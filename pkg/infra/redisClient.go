package infra

import (
	"context"
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
)

var (
	RedisClient *redis.Client
)

func init() {
	redisDb, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		BaseLogger.Sugar().Errorf("invalid redis db %v", err)
		return
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_HOST"),
		DB:   redisDb,
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			BaseLogger.Sugar().Infof("redis connected to host[%v] db[%v]", os.Getenv("REDIS_HOST"), redisDb)
			return nil
		},
	})

}
