//go:build wireinject
// +build wireinject

package main

import (
	"game-soul-technology/joker/joker-login-queue-server/pkg/client"
	"game-soul-technology/joker/joker-login-queue-server/pkg/config"
	"game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"game-soul-technology/joker/joker-login-queue-server/pkg/queue"

	"github.com/google/wire"
)

func Setup() (*Server, error) {
	wire.Build(wire.NewSet(
		ProvideServer,
		ProvideApplication,
		client.ProvideClientFactory,
		client.ProvideHub,
		config.ProvideQueueConfig,
		wire.Value(config.CFG),
		infra.ProvideHttpClient,
		infra.ProvideRedisClient,
		infra.ProvideLoggerFactory,
		queue.ProvideQueue,
		queue.ProvideStats,
	))
	return nil, nil
}
