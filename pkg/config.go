package main

import (
	"context"
	"game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"time"
)

type Config struct {
	OnlineUsers          uint `redis:"onlineUsers"`
	OnlineUsersThreshold uint `redis:"onlineUsersThreshold"`
	IsQueueEnabled       bool `redis:"isQueueEnabled"`
}

var (
	cfg = &Config{}
)

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
		logger.Infof("updating config")

		if _, err := infra.RedisClient.HSet(context.TODO(), cfgRedisKey, "onlineUsers", 1000).Result(); err != nil {
			logger.Errorf("err setting onlineUsers to redis %v", err)
			continue
		}

		if err := infra.RedisClient.HGetAll(context.TODO(), cfgRedisKey).Scan(c); err != nil {
			logger.Errorf("err reading config from redis %v", err)
			continue
		}
		logger.Infof("updated config[%+v]", c)
	}
}
