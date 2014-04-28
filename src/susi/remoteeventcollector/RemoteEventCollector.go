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
	"../apiserver"
	"../events"
	"encoding/json"
	"log"
	"net"
	"strings"
	//"../state"
)

type RemoteEventCollector struct {
	NewHostChan chan interface{}
	OwnNames    []string
}

func (ptr *RemoteEventCollector) HandleAwnsers(conn net.Conn) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	msg := new(apiserver.ApiMessage)
	for {
		if err := decoder.Decode(&msg); err != nil {
			log.Print(err)
			break
		}
		switch msg.Type {
		case "status":
			{
				log.Print(msg.Data.Payload)
			}
		case "event":
			{
				parts := strings.Split(msg.Data.Key, "@")
				key := parts[0]
				event := events.NewEvent(key, msg.Data.Payload)
				event.AuthLevel = msg.AuthLevel
				events.Publish(event)
			}
		}
	}
}

func (ptr *RemoteEventCollector) ConnectToHost(addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Print(err)
		return
	}
	go ptr.HandleAwnsers(conn)
	encoder := json.NewEncoder(conn)
	for _, name := range ptr.OwnNames {
		msg := new(apiserver.ApiMessage)
		msg.Type = "subscribe"
		msg.Data.Key = "*@" + name
		err = encoder.Encode(msg)
		if err != nil {
			log.Print(err)
			return
		}
	}
}

func Go() {
	ptr := new(RemoteEventCollector)
	ptr.OwnNames = []string{"all"}
	newHostChan, _ := events.Subscribe("hosts::new", 0)
	go func() {
		for event := range newHostChan {
			hostAddr := event.Payload.(string)
			go ptr.ConnectToHost(hostAddr)
		}
	}()
	return
}
