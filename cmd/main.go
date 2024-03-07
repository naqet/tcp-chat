package main

import "github.com/naqet/tcp-chat/network"

func main() {
	server := network.NewServer("localhost:8080")

	server.Start()
}
