package main

import (
	// "./events"
	// "./state"
	"./networking"
	"log"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	networking.StartTCPServer(uint16(12345))
	select {}
}
