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

package networking

import (
	"../events"
	"../state"
	"encoding/json"
	"log"
	"net"
	"flag"
)

var apiTcpPort = flag.String("apiTcpPort","4242","The port of the susi api server")

type subscribtionsType map[string]chan bool

type Connection struct {
	conn          net.Conn
	sender        *SyncedSender
	subscribtions subscribtionsType
}

func NewConnection(conn net.Conn) *Connection {
	connection := new(Connection)
	connection.conn = conn
	connection.sender = NewSyncedSender(conn)
	connection.subscribtions = make(subscribtionsType)
	return connection
}

func (conn *Connection) sendStatusMessage(id int64, key, msg string) {
	packet := new(ApiMessage)
	packet.Id = id
	packet.Type = "status"
	packet.Data.Key = key
	packet.Data.Payload = msg
	conn.sender.Send(packet)
}

func (conn *Connection) subscribe(req *ApiMessage) {
	topic := req.Data.Key
	if _, ok := conn.subscribtions[topic]; !ok {
		eventChan, unsubscribeChan := events.Subscribe(topic)
		closeChan := make(chan bool)
		conn.subscribtions[topic] = closeChan
		go func() {
			defer func() {
				unsubscribeChan <- true
			}()
			for {
				select {
				case event := <-eventChan:
					{
						resp := new(ApiMessage)
						resp.Id = req.Id
						resp.Type = "event"
						resp.Data.Key = topic
						resp.Data.Payload = event
						if evt,ok := event.(map[string]interface{}); ok {
							if topic,ok = evt["topic"].(string);ok {
								resp.Data.Key = topic
								if pay,ok := evt["payload"]; ok {
									resp.Data.Payload = pay
								}
							}
						}
						err := conn.sender.Send(resp)
						if err != nil {
							log.Print(err)
							return
						}
					}
				case <-closeChan:
					{
						return
					}
				}
			}
		}()
		conn.sendStatusMessage(req.Id,"ok", "successfully subscribed to "+topic)
	} else {
		conn.sendStatusMessage(req.Id,"error", "you are allready subscribed to "+topic)
		log.Print("allready subscribed to topic ", topic)
	}
}

func (conn *Connection) unsubscribe(req *ApiMessage) {
	topic := req.Data.Key
	if ch, ok := conn.subscribtions[topic]; ok {
		ch <- true
		delete(conn.subscribtions, topic)
		conn.sendStatusMessage(req.Id,"ok", "successfully unsubscribed from "+topic)
	} else {
		conn.sendStatusMessage(req.Id,"error", "you are not subscribed to "+topic)
		log.Print("not subscribed to topic ", topic)
	}
}

type ApiMessage struct {
	Id   int64  `json:"id,omitempty"`
	Type string `json:"type"`
	Data struct {
		Key     string      `json:"key"`
		Payload interface{} `json:"payload"`
	} `json:"data"`
}

func HandleConnection(conn net.Conn) {
	connection := NewConnection(conn)
	defer func() {
		for _, ch := range connection.subscribtions {
			ch <- true
		}
		connection.sender.Close()
		conn.Close()
	}()
	decoder := json.NewDecoder(conn)
	for {
		req := ApiMessage{}
		err := decoder.Decode(&req)
		if err != nil {
			log.Print(err)
			return
		}
		switch req.Type {
		case "subscribe":
			{
				connection.subscribe(&req)
			}
		case "unsubscribe":
			{
				connection.unsubscribe(&req)
			}
		case "publish":
			{
				events.Publish(req.Data.Key, req.Data.Payload)
				connection.sendStatusMessage(req.Id,"ok", "successfully published event to "+req.Data.Key)
			}
		case "set":
			{
				state.Set(req.Data.Key,req.Data.Payload)
				connection.sendStatusMessage(req.Id,"ok", "successfully saved data to "+req.Data.Key)
			}
		case "push":
			{
				state.Push(req.Data.Key,req.Data.Payload)
				connection.sendStatusMessage(req.Id,"ok", "successfully pushed data to "+req.Data.Key)
			}
		case "enqueue":
			{
				state.Enqueue(req.Data.Key,req.Data.Payload)
				connection.sendStatusMessage(req.Id,"ok", "successfully queued data to "+req.Data.Key)
			}
		case "get":
			{
				data := state.Get(req.Data.Key)
				packet := new(ApiMessage)
				packet.Id = req.Id
				packet.Type = "response"
				packet.Data.Key = req.Data.Key
				packet.Data.Payload = data
				connection.sender.Send(packet)
			}
		case "pop":
			{
				data := state.Pop(req.Data.Key)
				packet := new(ApiMessage)
				packet.Id = req.Id
				packet.Type = "response"
				packet.Data.Key = req.Data.Key
				packet.Data.Payload = data
				connection.sender.Send(packet)
			}
		case "dequeue":
			{
				data := state.Dequeue(req.Data.Key)
				packet := new(ApiMessage)
				packet.Id = req.Id
				packet.Type = "response"
				packet.Data.Key = req.Data.Key
				packet.Data.Payload = data
				connection.sender.Send(packet)
			}
		case "unset":
			{
				state.Unset(req.Data.Key)
				connection.sendStatusMessage(req.Id,"ok", "successfully unset data from "+req.Data.Key)
			}
		default:
			{
				connection.sendStatusMessage(req.Id,"error", "no such request type: "+req.Type)	
			}
		}
		// fmt.Println("Request: ",req)
	}
}

func StartTCPServer() error {
	portStr := state.Get("apiTcpPort").(string)
	listener, err := net.Listen("tcp", ":"+portStr)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Print(err)
				continue
			}
			go HandleConnection(conn)
		}
	}()
	log.Print("successfully started susi api server on ",listener.Addr())
	return nil
}
