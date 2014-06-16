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

package remoteeventcollector

import (
	"github.com/trusch/susi/apiserver"
	"github.com/trusch/susi/config"
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/session"
	"github.com/trusch/susi/state"
	"log"
	"testing"
	"time"
)

func TestRemoteEventCollector(t *testing.T) {
	events.Go()
	state.Go()
	config.Go()
	session.Go()

	state.Set("apiserver.port", "12345")
	state.Set("apiserver.tls.port", "")
	state.Set("apiserver.tls.cert", "")
	state.Set("apiserver.tls.key", "")

	apiserver.Go()

	New([]string{"foo"})

	testChan, _ := events.Subscribe("*", 0)
	go func() {
		for event := range testChan {
			log.Print(event)
		}
	}()

	event := events.NewEvent("hosts::new", "localhost:12345")
	event.AuthLevel = 0
	events.Publish(event)

	ch, _ := events.Subscribe("test", 0)

	time.Sleep(100 * time.Millisecond)
	event = events.NewEvent("test@foo", nil)
	events.Publish(event)

	event = <-ch
	if event.Topic != "test" {
		t.Error("wrong event")
	}

}
