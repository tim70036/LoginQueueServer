package config

import (
	"context"
	"game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/imroc/req/v3"
	"go.uber.org/zap"
)

type QueueConfig struct {
	// Current online users number from main server.
	OnlineUsers uint `redis:"onlineUsers"`

	// Max allowed online users number.
	OnlineUsersThreshold uint `redis:"onlineUsersThreshold"`

	// Percentage of OnlineUsersThreshold that will start queueing.
	// For example, if OnlineUsersThreshold is 1000 and
	// StartQueueThreshold is 80%, queue will start functioning when
	// OnlineUsers reaches 1000 x 80% = 800. Will stop functioning
	// when OnlineUsers drops below 800. Default to 100%.
	StartQueueThreshold float32 `redis:"startQueueThreshold"`

	// If false, will not queue no matter what.
	IsQueueEnabled bool `redis:"isQueueEnabled"`

	FreeSlots     uint
	freeSlotsLock sync.Mutex

	redisClient *redis.Client
	httpClient  *req.Client
	logger      *zap.SugaredLogger
}

func ProvideQueueConfig(redisClient *redis.Client, httpClient *req.Client, loggerFactory *infra.LoggerFactory) *QueueConfig {
	return &QueueConfig{
		StartQueueThreshold: 1,
		redisClient:         redisClient,
		httpClient:          httpClient,
		logger:              loggerFactory.Create("QueueConfig").Sugar(),
	}
}

const (
	// Update config with this interval.
	cfgUpdateInterval = 5 * time.Second

	// QueueConfig redis key.
	cfgRedisKey = "config"
)

func (c *QueueConfig) ShouldQueue() bool {
	return c.IsQueueEnabled &&
		(float32(c.OnlineUsers) >= float32(c.OnlineUsersThreshold)*c.StartQueueThreshold)
}

func (c *QueueConfig) ReplenishFreeSlots() {
	c.freeSlotsLock.Lock()
	defer c.freeSlotsLock.Unlock()

	var newFreeSlots uint = 0
	if c.OnlineUsers < c.OnlineUsersThreshold {
		newFreeSlots = c.OnlineUsersThreshold - c.OnlineUsers
	}

	c.FreeSlots = newFreeSlots

	c.logger.Infof("replenish freeSlots[%v]", c.FreeSlots)
}

func (c *QueueConfig) TakeOneSlot() bool {
	c.freeSlotsLock.Lock()
	defer c.freeSlotsLock.Unlock()

	if c.FreeSlots <= 0 {
		return false
	}

	c.FreeSlots--
	return true
}

func (c *QueueConfig) Run() {
	ticker := time.NewTicker(cfgUpdateInterval)
	for ; true; <-ticker.C {
		c.logger.Infof("updating config")

		if err := c.redisClient.HGetAll(context.TODO(), cfgRedisKey).Scan(c); err != nil {
			c.logger.Errorf("err reading config from redis %v", err)
			continue
		}

		c.logger.Infof("will queue if online users reach %+v", float32(c.OnlineUsersThreshold)*c.StartQueueThreshold)

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

		newOnlineUsers, err := strconv.Atoi(onlineResult.Data.OnlineUsers)
		if err != nil {
			c.logger.Errorf("cannot parse online user number[%v] to int %v", newOnlineUsers, err)
			continue
		}

		// Will skip if main server hasn't updated his online user
		// number. We must do this in case that main server do not
		// update frequently. In this case, queue server will dequeue
		// too many users in a short period of time.
		if newOnlineUsers == int(c.OnlineUsers) {
			c.logger.Infof("skip update since onlineUsers not change config[%+v]", c)
			continue
		}

		c.OnlineUsers = uint(newOnlineUsers)
		c.ReplenishFreeSlots()

		if _, err := c.redisClient.HSet(context.TODO(), cfgRedisKey,
			"onlineUsers", c.OnlineUsers,
		).Result(); err != nil {
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
