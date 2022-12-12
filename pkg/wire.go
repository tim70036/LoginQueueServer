//go:build wireinject
// +build wireinject

package main

import "github.com/google/wire"

func Setup() *Server {
	wire.Build(wire.NewSet(ProvideServer, ProvideApplication, ProvideConfig, ProvideHub, ProvideQueue, ProvideStats))
	return nil
}
