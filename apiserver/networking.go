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

package apiserver

import (
	"code.google.com/p/go-uuid/uuid"
	"crypto/tls"

	"encoding/json"
	"flag"
	"github.com/trusch/susi/authentification"
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/state"
	"log"
	"net"
	"time"
)

var apiTcpPort = flag.String("apiserver.port", "4000", "The port of the susi api server")
var apiTlsPort = flag.String("apiserver.tls.port", "4001", "The port of the susi api server")
var apiCertFile = flag.String("apiserver.tls.cert", "", "The certificate to use in the api server")
var apiKeyFile = flag.String("apiserver.tls.key", "", "The key to use in the api server")

type ApiMessage struct {
	Id         int64       `json:"id,omitempty"`
	AuthLevel  uint8       `json:"authlevel,omitempty"`
	Type       string      `json:"type"`
	Key        string      `json:"key"`
	ReturnAddr string      `json:"returnaddr,omitempty"`
	Payload    interface{} `json:"payload,omitempty"`
}

func NewApiMessage() *ApiMessage {
	msg := new(ApiMessage)
	msg.AuthLevel = 255
	return msg
}

type subscribtionsType map[string]chan bool

type Connection struct {
	conn          net.Conn
	sender        *SyncedSender
	subscribtions subscribtionsType
	username      string
	authlevel     uint8
}

func NewConnection(conn net.Conn) *Connection {
	connection := new(Connection)
	connection.conn = conn
	connection.sender = NewSyncedSender(conn)
	connection.subscribtions = make(subscribtionsType)
	connection.authlevel = 3
	connection.username = "anonymous"
	return connection
}

func (conn *Connection) sendStatusMessage(id int64, key, msg string) {
	packet := NewApiMessage()
	packet.AuthLevel = 0
	packet.Id = id
	packet.Type = "status"
	packet.Key = key
	packet.Payload = msg
	conn.sender.Send(packet)
}

func (conn *Connection) subscribe(req *ApiMessage) {
	topic := req.Key
	if _, ok := conn.subscribtions[topic]; !ok {
		eventChan, unsubscribeChan := events.Subscribe(topic, req.AuthLevel)
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
						resp := NewApiMessage()
						log.Print(resp)
						resp.AuthLevel = event.AuthLevel
						resp.Id = req.Id
						resp.Type = "event"
						resp.Key = event.Topic
						resp.Payload = event.Payload
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
		conn.sendStatusMessage(req.Id, "ok", "successfully subscribed to "+topic)
	} else {
		conn.sendStatusMessage(req.Id, "error", "you are allready subscribed to "+topic)
	}
}

func (conn *Connection) unsubscribe(req *ApiMessage) {
	topic := req.Key
	if ch, ok := conn.subscribtions[topic]; ok {
		ch <- true
		delete(conn.subscribtions, topic)
		conn.sendStatusMessage(req.Id, "ok", "successfully unsubscribed from "+topic)
	} else {
		conn.sendStatusMessage(req.Id, "error", "you are not subscribed to "+topic)
	}
}

func (conn *Connection) checkUser(username, password string) bool {
	awnserTopic := uuid.New()
	awnserChan, closeChan := events.Subscribe(awnserTopic, 0)
	event := events.NewEvent("authentification::checkuser", map[string]interface{}{
		"username": username,
		"password": password,
	})
	event.ReturnAddr = awnserTopic
	event.AuthLevel = 0
	events.Publish(event)
	awnser_ := <-awnserChan
	closeChan <- true
	awnser := awnser_.Payload.(*authentification.AwnserData)
	if awnser.Success {
		user := awnser.Message.(*authentification.User)
		conn.username = user.Username
		conn.authlevel = user.AuthLevel
		return true
	}
	return false
}

func HandleConnection(conn net.Conn, authlevel uint8) {
	conn.SetDeadline(time.Time{})
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
			log.Print("lost connection or parse error: ", err)
			return
		}
		if req.AuthLevel < authlevel {
			req.AuthLevel = authlevel
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
				if req.Key == "controller::auth::info" {
					connection.sendStatusMessage(req.Id, "ok", "successfully published event to "+req.Key)
					event := events.NewEvent(req.ReturnAddr, map[string]interface{}{
						"error":    false,
						"username": connection.username,
					})
					event.AuthLevel = connection.authlevel
					events.Publish(event)
					break
				}
				if req.Key == "controller::auth::login" {
					connection.sendStatusMessage(req.Id, "ok", "successfully published event to "+req.Key)
					payload, ok := req.Payload.(map[string]interface{})
					if !ok {
						event := events.NewEvent(req.ReturnAddr, map[string]interface{}{
							"error":   true,
							"message": "malformed payload",
						})
						event.AuthLevel = connection.authlevel
						events.Publish(event)
						break
					}
					username, ok1 := payload["username"].(string)
					password, ok2 := payload["password"].(string)
					if !ok1 || !ok2 {
						event := events.NewEvent(req.ReturnAddr, map[string]interface{}{
							"error":   true,
							"message": "malformed payload",
						})
						event.AuthLevel = connection.authlevel
						events.Publish(event)
						break
					}
					if connection.checkUser(username, password) {
						event := events.NewEvent(req.ReturnAddr, map[string]interface{}{
							"error":    false,
							"username": connection.username,
						})
						event.AuthLevel = req.AuthLevel
						events.Publish(event)
					} else {
						event := events.NewEvent(req.ReturnAddr, map[string]interface{}{
							"error":   true,
							"message": "wrong username/password",
						})
						event.AuthLevel = connection.authlevel
						events.Publish(event)
					}
					break
				}
				if req.Key == "controller::auth::logout" {
					connection.sendStatusMessage(req.Id, "ok", "successfully published event to "+req.Key)
					connection.username = "anonymous"
					connection.authlevel = 3
					event := events.NewEvent(req.ReturnAddr, map[string]interface{}{
						"error":    false,
						"username": connection.username,
					})
					event.AuthLevel = connection.authlevel
					events.Publish(event)
					break
				}
				event := events.NewEvent(req.Key, req.Payload)
				event.AuthLevel = req.AuthLevel
				event.ReturnAddr = req.ReturnAddr
				found := events.Publish(event)
				if found {
					connection.sendStatusMessage(req.Id, "ok", "successfully published event to "+req.Key)
				} else {
					connection.sendStatusMessage(req.Id, "error", "nobody is subscribed to "+req.Key)
				}
			}
		case "set":
			{
				state.Set(req.Key, req.Payload)
				connection.sendStatusMessage(req.Id, "ok", "successfully saved data to "+req.Key)
			}
		case "push":
			{
				state.Push(req.Key, req.Payload)
				connection.sendStatusMessage(req.Id, "ok", "successfully pushed data to "+req.Key)
			}
		case "enqueue":
			{
				state.Enqueue(req.Key, req.Payload)
				connection.sendStatusMessage(req.Id, "ok", "successfully queued data to "+req.Key)
			}
		case "get":
			{
				data := state.Get(req.Key)
				packet := new(ApiMessage)
				packet.Id = req.Id
				packet.Type = "response"
				packet.Key = req.Key
				packet.Payload = data
				connection.sender.Send(packet)
			}
		case "pop":
			{
				data := state.Pop(req.Key)
				packet := new(ApiMessage)
				packet.Id = req.Id
				packet.Type = "response"
				packet.Key = req.Key
				packet.Payload = data
				connection.sender.Send(packet)
			}
		case "dequeue":
			{
				data := state.Dequeue(req.Key)
				packet := new(ApiMessage)
				packet.Id = req.Id
				packet.Type = "response"
				packet.Key = req.Key
				packet.Payload = data
				connection.sender.Send(packet)
			}
		case "unset":
			{
				state.Unset(req.Key)
				connection.sendStatusMessage(req.Id, "ok", "successfully unset data from "+req.Key)
			}
		case "login":
			{
				username := req.Key
				password, ok := req.Payload.(string)
				if !ok {
					connection.sendStatusMessage(req.Id, "error", "no password provided")
				}
				if connection.checkUser(username, password) {
					connection.sendStatusMessage(req.Id, "ok", "successfully logged in as "+username)
				} else {
					connection.sendStatusMessage(req.Id, "error", "failed logging in as "+username)
				}
			}
		case "logout":
			{
				connection.username = "anonymous"
				connection.authlevel = 3
				connection.sendStatusMessage(req.Id, "ok", "successfully logged out")
			}
		default:
			{
				connection.sendStatusMessage(req.Id, "error", "no such request type: "+req.Type)
			}
		}
	}
}

func Go() {
	portStr := state.Get("apiserver.port").(string)
	tlsPortStr := state.Get("apiserver.tls.port").(string)
	certStr := state.Get("apiserver.tls.cert").(string)
	keyStr := state.Get("apiserver.tls.key").(string)

	if certStr != "" && keyStr != "" {
		cert, err := tls.LoadX509KeyPair(certStr, keyStr)
		if err != nil {
			log.Fatal(err)
		}
		config := &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequireAnyClientCert,
		}

		listener, err := tls.Listen("tcp", ":"+tlsPortStr, config)
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
				tlsConn := conn.(*tls.Conn)
				err = tlsConn.Handshake()
				if err != nil {
					log.Print(err)
					continue
				}
				myCert := cert.Certificate[0]
				peerCertIsMyCert := true
				if certs := tlsConn.ConnectionState().PeerCertificates; len(certs) > 0 {
					peerCert := certs[0].Raw
					if len(peerCert) == len(myCert) {
						for idx, chr := range peerCert {
							if chr != myCert[idx] {
								peerCertIsMyCert = false
								break
							}
						}
					}
				}
				if peerCertIsMyCert {
					go HandleConnection(conn, 0)
				} else {
					go HandleConnection(conn, 1)
				}
			}
		}()
		log.Print("successfully started susi tls api server on ", listener.Addr())
	}

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
			go HandleConnection(conn, 3)
		}
	}()
	log.Print("successfully started susi api server on ", listener.Addr())
	return
}
