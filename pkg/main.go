package main

import (
	"log"
)

func main() {
	server, err := Setup()
	if err != nil {
		log.Fatalf("main start failed %v", err)
		return
	}

	server.Run()
}
