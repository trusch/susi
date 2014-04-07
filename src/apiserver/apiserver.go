/*
 * Copyright (c) 2014, webvariants GmbH, http://www.webvariants.de
 *
 * This file is released under the terms of the MIT license. You can find the
 * complete text in the attached LICENSE file or online at:
 *
 * http://www.opensource.org/licenses/mit-license.php
 * 
 * @author: Tino Rusch (tino.rusch@webvariants.de)
 */

package main

import (
	"./events"
	"./config"
	"./networking"
	"./autodiscovery"
	"./remoteeventcollector"
	"log"
)

func EventPrinter() {
	ch,_ := events.Subscribe("*");
	go func(){
		for evt := range ch {
			log.Println(evt)
		}
	}()
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	config.NewManager()
	EventPrinter();
	networking.StartTCPServer()
	remoteeventcollector.New()
	autodiscovery.Run()

	select {}
}
