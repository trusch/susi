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

import(
	"../events"
	"../networking"
	"net"
	"encoding/json"
	"log"
	"strings"
	//"../state"
)

type RemoteEventCollector struct {
	NewHostChan chan interface{}
	OwnNames []string
}

func (ptr *RemoteEventCollector) HandleAwnsers(conn net.Conn) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	msg := new(networking.ApiMessage)	
	for {
		if err := decoder.Decode(&msg); err!=nil {
			log.Print(err)
			break
		}
		switch msg.Type {
			case "status": {
				log.Print(msg.Data.Payload)
			}
			case "event": { 
				parts := strings.Split(msg.Data.Key,"@")
				key := parts[0]
				events.Publish(key,msg.Data.Payload)
			}
		}
	}
}

func (ptr *RemoteEventCollector) ConnectToHost(addr string){
	conn,err := net.Dial("tcp",addr)
	if err!=nil {
		log.Print(err)
		return
	}
	go ptr.HandleAwnsers(conn)
	encoder := json.NewEncoder(conn)
	for _,name := range ptr.OwnNames {
		msg := new(networking.ApiMessage)
		msg.Type = "subscribe"
		msg.Data.Key = "*@"+name 
		err = encoder.Encode(msg)
		if err!=nil {
			log.Print(err)
			return
		}
	}
}

func New() *RemoteEventCollector {
	ptr := new(RemoteEventCollector)
	ptr.OwnNames = []string{"all"}
	hCh,_ := events.Subscribe("hosts::new");
	ptr.NewHostChan = hCh
	go func(){
		for event := range ptr.NewHostChan {
			hostAddr := event.(string)
			go ptr.ConnectToHost(hostAddr);
		}
	}()
	return ptr
}