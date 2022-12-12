package infra

import (
	"context"
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
)

func ProvideRedisClient(loggerFactory *LoggerFactory) (*redis.Client, error) {
	logger := loggerFactory.Create("RedisClient").Sugar()
	redisDb, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		logger.Errorf("invalid redis db %v", err)
		return nil, err
	}

	return redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_HOST"),
		DB:   redisDb,
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			logger.Infof("redis connected to host[%v] db[%v]", os.Getenv("REDIS_HOST"), redisDb)
			return nil
		},
	}), nil
}
