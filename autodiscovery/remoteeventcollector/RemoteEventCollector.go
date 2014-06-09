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
	"encoding/json"
	"github.com/trusch/susi/apiserver"
	"github.com/trusch/susi/events"
	"net"
	"strings"
)

type RemoteEventCollector struct {
	NewHostChan chan *events.Event
	OwnNames    []string
}

func New(names []string) *RemoteEventCollector {
	ptr := new(RemoteEventCollector)
	ptr.OwnNames = []string{"all"}
	if names != nil {
		ptr.OwnNames = append(ptr.OwnNames, names...)
	}
	ptr.NewHostChan, _ = events.Subscribe("hosts::new", 0)
	go func() {
		for event := range ptr.NewHostChan {
			hostAddr := event.Payload.(string)
			go ptr.ConnectToHost(hostAddr)
		}
	}()
	return ptr
}

func (ptr *RemoteEventCollector) ConnectToHost(addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return
	}
	go ptr.HandleAwnsers(conn, addr)
	encoder := json.NewEncoder(conn)
	for _, name := range ptr.OwnNames {
		msg := new(apiserver.ApiMessage)
		msg.Type = "subscribe"
		msg.Key = "*@" + name
		err = encoder.Encode(msg)
		if err != nil {
			return
		}
	}
}

func (ptr *RemoteEventCollector) HandleAwnsers(conn net.Conn, addr string) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	msg := new(apiserver.ApiMessage)
	for {
		if err := decoder.Decode(&msg); err != nil {
			event := events.NewEvent("hosts::lost", addr)
			event.AuthLevel = 0
			events.Publish(event)
			break
		}
		switch msg.Type {
		case "status":
			{
			}
		case "event":
			{
				parts := strings.Split(msg.Key, "@")
				key := parts[0]
				event := events.NewEvent(key, msg.Payload)
				event.AuthLevel = msg.AuthLevel
				events.Publish(event)
			}
		}
	}
}
