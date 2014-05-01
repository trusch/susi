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
	"./apiserver"
	"./autodiscovery"
	"./config"
	"./events"
	"./remoteeventcollector"
	"./state"
	"./webstack"
	"flag"
	"log"
)

func EventPrinter() {
	ch, _ := events.Subscribe("*", 0)
	go func() {
		for evt := range ch {
			log.Println("EVENT: ", evt)
		}
	}()
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	flag.Parse()

	events.Go()

	EventPrinter()

	state.Go()
	config.Go()
	apiserver.Go()
	remoteeventcollector.Go()
	autodiscovery.Go()
	webstack.Go()
	select {}
}
