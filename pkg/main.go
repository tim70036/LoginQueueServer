package main

import (
	"game-soul-technology/joker/joker-login-queue-server/pkg/infra"
	"log"

	"go.uber.org/zap/zapcore"
)

func main() {
	// TODO: remove this
	infra.LoggerLevel.SetLevel(zapcore.DebugLevel)
	// infra.HttpClient.EnableDumpAll()

	server, err := Setup()
	if err != nil {
		log.Fatalf("main start failed %v", err)
		return
	}

	server.Run()
}
