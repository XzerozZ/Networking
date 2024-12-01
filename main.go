package main

import (
	"os"
	"networking/internal/api"
)

func main() {
	nodeID := os.Getenv("NODE_ID")
	listenAddr := os.Getenv("LISTEN_ADDR")

	if nodeID == "" || listenAddr == "" {
		panic("NODE_ID and LISTEN_ADDR environment variables are required")
	}

	server := api.NewServer(nodeID, listenAddr)
	server.Run()
}
