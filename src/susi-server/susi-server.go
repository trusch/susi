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
	"./jsengine"
	"./remoteeventcollector"
	"./state"
	"./webstack"
	"flag"
	"log"

	"./controller/firebirdconnector"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	flag.Parse()

	events.Go()
	
	go func(){
		ch,_ := events.Subscribe("*",0)
		for event := range ch {
			log.Print(event)
		}
	}()
	
	state.Go()
	config.Go()
	apiserver.Go()
	remoteeventcollector.Go()
	autodiscovery.Go()
	webstack.Go()
	jsengine.Go()

	firebirdconnector.Go()


	event := events.NewEvent("firebird::query",map[string]interface{}{
		"query": "SELECT DISTINCT JOB_TITLE FROM JOB WHERE JOB_TITLE LIKE ?;",
		"args" : []interface{}{
			"%er",
		},
	})
	event.AuthLevel = 0
	event.ReturnAddr = "sample::awnser"
	events.Publish(event)

	select {}
}
