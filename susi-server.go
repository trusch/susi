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
	"github.com/golang/glog"
	"github.com/trusch/susi/apiserver"
	"github.com/trusch/susi/authentification"
	"github.com/trusch/susi/autodiscovery"
	"github.com/trusch/susi/config"
	"github.com/trusch/susi/controller/firebirdconnector"
	"github.com/trusch/susi/enginestarter"
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/jsengine"
	"github.com/trusch/susi/session"
	"github.com/trusch/susi/state"
	"github.com/trusch/susi/webstack"
	//"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var logfile = flag.String("logger.file", "", "where to write logs")

func setupLogger() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func main() {
	defer func() { glog.Flush() }()
	flag.Parse()
	glog.Info("start main")

	setupLogger()

	events.Go()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-c
		event := events.NewEvent("global::shutdown", nil)
		event.AuthLevel = 0
		events.Publish(event)
		time.Sleep(1 * time.Second)
		os.Exit(1)
	}()

	/*	go func() {
		ch, _ := events.Subscribe("*", 0)
		for event := range ch {
			log.Print(event)
		}
	}()*/

	state.Go()
	config.Go()
	session.Go()
	apiserver.Go()
	autodiscovery.Go()
	authentification.Go()
	webstack.Go()
	firebirdconnector.Go()
	jsengine.Go()
	enginestarter.Go()

	select {}
}
