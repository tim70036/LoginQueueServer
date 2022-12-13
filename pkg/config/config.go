package config

import (
	"context"
	"game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/imroc/req/v3"
	"go.uber.org/zap"
)

type Config struct {
	OnlineUsers          uint `redis:"onlineUsers"`
	OnlineUsersThreshold uint `redis:"onlineUsersThreshold"`
	IsQueueEnabled       bool `redis:"isQueueEnabled"`

	redisClient *redis.Client
	httpClient  *req.Client
	logger      *zap.SugaredLogger
}

func ProvideConfig(redisClient *redis.Client, httpClient *req.Client, loggerFactory *infra.LoggerFactory) *Config {
	return &Config{
		redisClient: redisClient,
		httpClient:  httpClient,
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
	if slots := c.OnlineUsersThreshold - c.OnlineUsers; slots > 0 {
		return slots
	}
	return 0
}

func (c *Config) Run() {
	ticker := time.NewTicker(cfgUpdateInterval)
	for ; true; <-ticker.C {
		c.logger.Infof("updating config")

		onlineResult := &struct {
			Data struct {
				OnlineUsers string `json:"onlineUsers"`
				PlayingAis  string `json:"playingAis"`
			} `json:"data"`
		}{}

		resp, err := c.httpClient.R().
			SetHeader("jtoken", os.Getenv("MAIN_SERVER_API_KEY")).
			SetResult(onlineResult).
			Get(os.Getenv("MAIN_SERVER_HOST") + "/queue/online-users")

		if err != nil {
			c.logger.Errorf("request failed %v", err)
			continue
		}

		if resp.IsError() {
			c.logger.Errorf("request failed with status[%v]", resp.Status)
			continue
		}

		c.logger.Infof("retrieved online user result[%+v]", onlineResult)

		number, err := strconv.Atoi(onlineResult.Data.OnlineUsers)
		if err != nil {
			c.logger.Errorf("cannot parse online user number[%v] to int %v", number, err)
			continue
		}

		if _, err := c.redisClient.HSet(context.TODO(), cfgRedisKey, "onlineUsers", number).Result(); err != nil {
			c.logger.Errorf("err setting onlineUsers to redis %v", err)
			continue
		}

		// TODO: remove this
		if _, err := c.redisClient.HSet(context.TODO(), cfgRedisKey, "onlineUsersThreshold", number+1).Result(); err != nil {
			c.logger.Errorf("err setting onlineUsersThreshold to redis %v", err)
			continue
		}

		if err := c.redisClient.HGetAll(context.TODO(), cfgRedisKey).Scan(c); err != nil {
			c.logger.Errorf("err reading config from redis %v", err)
			continue
		}
		c.logger.Infof("updated config[%+v]", c)
	}
}
