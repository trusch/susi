package main

import (
	// "./events"
	// "./state"
	"./networking"
	"./autodiscovery"
	"log"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	networking.StartTCPServer(uint16(12345))
	autodiscovery.Run()
	select {}
}
