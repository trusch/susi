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
	"flag"
	"github.com/trusch/susi/apiserver"
	"github.com/trusch/susi/autodiscovery"
	"github.com/trusch/susi/config"
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/jsengine"
	"github.com/trusch/susi/state"
	"github.com/trusch/susi/webstack"
	"log"

	"github.com/trusch/susi/controller/firebirdconnector"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	flag.Parse()

	events.Go()

	go func() {
		ch, _ := events.Subscribe("*", 0)
		for event := range ch {
			log.Print(event)
		}
	}()

	state.Go()
	config.Go()
	apiserver.Go()
	autodiscovery.Go()
	webstack.Go()
	firebirdconnector.Go()
	jsengine.Go()

	select {}
}
