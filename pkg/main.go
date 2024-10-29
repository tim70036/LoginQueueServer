package main

import (
	"flag"
	"io"
	"log"
	"os"
)

func main() {
	flag.CommandLine.SetOutput(io.Writer(os.Stdout))
	flag.PrintDefaults()
	flag.Parse()

	server, err := Setup()
	if err != nil {
		log.Fatalf("main start failed %v", err)
		return
	}

	server.Run()
}
