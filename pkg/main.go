package main

import (
	"game-soul-technology/joker/joker-login-queue-server/pkg/infra"

	"go.uber.org/zap/zapcore"
)

var (
	logger = infra.BaseLogger.Sugar()
)

func main() {
	defer logger.Sync()

	// TODO: remove this
	infra.LoggerLevel.SetLevel(zapcore.DebugLevel)
	infra.HttpClient.EnableDumpAll()

	server := Setup()
	server.Run()
}
