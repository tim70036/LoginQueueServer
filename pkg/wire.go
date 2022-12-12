//go:build wireinject
// +build wireinject

package main

import (
	"game-soul-technology/joker/joker-login-queue-server/pkg/client"
	"game-soul-technology/joker/joker-login-queue-server/pkg/config"
	"game-soul-technology/joker/joker-login-queue-server/pkg/queue"

	"github.com/google/wire"
)

func Setup() *Server {
	wire.Build(wire.NewSet(
		ProvideServer,
		ProvideApplication,
		config.ProvideConfig,
		client.ProvideClientFactory,
		client.ProvideHub,
		queue.ProvideQueue,
		queue.ProvideStats,
	))
	return nil
}
