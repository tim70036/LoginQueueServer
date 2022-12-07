package main

import (
	"time"
)

type Config struct {
	onlineUsers          uint
	onlineUsersThreshold uint
	isQueueEnabled       bool
}

var (
	config = NewConfig()
)

const (
	// Update config with this interval.
	configUpdateInterval = 10 * time.Second
)

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) GetFreeSlots() uint {
	// TODO: race condition?
	if slots := c.onlineUsers - c.onlineUsersThreshold; slots > 0 {
		return slots
	}
	return 0
}

func (c *Config) Run() {
	ticker := time.NewTicker(configUpdateInterval)
	for ; true; <-ticker.C {
		// TODO: get data from redis and main server.
		c.onlineUsers = 1
		c.onlineUsersThreshold = 2
	}
}
