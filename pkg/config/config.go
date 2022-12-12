package config

import (
	"context"
	"game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type Config struct {
	OnlineUsers          uint `redis:"onlineUsers"`
	OnlineUsersThreshold uint `redis:"onlineUsersThreshold"`
	IsQueueEnabled       bool `redis:"isQueueEnabled"`

	redisClient *redis.Client
	logger      *zap.SugaredLogger
}

func ProvideConfig(redisClient *redis.Client, loggerFactory *infra.LoggerFactory) *Config {
	return &Config{
		redisClient: redisClient,
		logger:      loggerFactory.Create("Config").Sugar(),
	}
}

const (
	// Update config with this interval.
	cfgUpdateInterval = 30 * time.Second

	// Config redis key.
	cfgRedisKey = "config"
)

func (c *Config) GetFreeSlots() uint {
	// TODO: race condition?
	if slots := c.OnlineUsersThreshold - c.OnlineUsers; slots > 0 {
		return slots
	}
	return 0
}

func (c *Config) Run() {
	ticker := time.NewTicker(cfgUpdateInterval)
	for ; true; <-ticker.C {
		// TODO: get data from redis and main server.
		c.logger.Infof("updating config")

		if _, err := c.redisClient.HSet(context.TODO(), cfgRedisKey, "onlineUsers", 1000).Result(); err != nil {
			c.logger.Errorf("err setting onlineUsers to redis %v", err)
			continue
		}

		if err := c.redisClient.HGetAll(context.TODO(), cfgRedisKey).Scan(c); err != nil {
			c.logger.Errorf("err reading config from redis %v", err)
			continue
		}
		c.logger.Infof("updated config[%+v]", c)
	}
}
