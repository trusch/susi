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

package events

/*
This provides a Publish-Subscribe server for the system
*/

import (
	"log"
	"time"
	"errors"
	"path/filepath"
	"strings"
)

type CommandType uint8

const (
	SUBSCRIBE CommandType = iota
	UNSUBSCRIBE
	PUBLISH
)

type EventSystem struct {
	cmdChan chan *command
	topics  map[string]map[int64]chan interface{}
	globs map[int64]*globChan
}

type globChan struct {
	EventChan chan interface{}
	Glob string
}

func isGlob(pattern string) bool {
	return strings.IndexAny(pattern, "*?[") >= 0
}

type command struct {
	Type    CommandType
	Topic   string
	Payload interface{}
	Result  chan interface{}
}

type unsubscribeResult struct {
	EventChan chan interface{}
	CloseChan chan bool
}

func (ptr *EventSystem) subscribe(topic string) (eventChannel chan interface{},closeChannel chan bool) {
	eventChannel = make(chan interface{},10)
	closeChannel = make(chan bool)
	now := time.Now().UnixNano();
	if(isGlob(topic)){
		glob := new(globChan)
		glob.EventChan = eventChannel
		glob.Glob = topic
		ptr.globs[now] = glob
	}else{
		channelMap := ptr.topics[topic]
		if channelMap==nil {
			tmp := make(map[int64]chan interface{})
			ptr.topics[topic] = tmp
			channelMap = tmp
		}
		channelMap[now] = eventChannel
	}
	//log.Print("subscribed to ",topic," (",now,")")
	go func(){
		<-closeChannel
		//log.Print("unsubscribed from ",topic," (",now,")")
		ptr.cmdChan <- &command{
			Type: UNSUBSCRIBE,
			Topic: topic,
			Payload: now,
		}
	}()
	return eventChannel,closeChannel
}

var eventSystem *EventSystem

func init() {
	eventSystem = new(EventSystem)
	eventSystem.cmdChan = make(chan *command, 10)
	eventSystem.topics = make(map[string]map[int64]chan interface{})
	eventSystem.globs = make(map[int64]*globChan)
	go func() {
		for cmd := range eventSystem.cmdChan {
			switch cmd.Type {
			case SUBSCRIBE:
				{
					ec,cc := eventSystem.subscribe(cmd.Topic)
					cmd.Result <- unsubscribeResult{
						EventChan: ec,
						CloseChan: cc,
					}
				}
			case PUBLISH:
				{
					chans := eventSystem.topics[cmd.Topic]
					found := false
					for _,glob := range eventSystem.globs {
						if ok,err := filepath.Match(glob.Glob,cmd.Topic); ok && err==nil {
							found = true
							event := make(map[string]interface{})
							event["payload"] = cmd.Payload
							event["topic"] = cmd.Topic
							glob.EventChan <- event
						}
					}
					for _, outChan := range chans {
						outChan <- cmd.Payload
						found = true
					}
					cmd.Result <- found
				}
			case UNSUBSCRIBE:
				{
					topic := cmd.Topic
					id := cmd.Payload.(int64)
					if topic!="" {
						chans := eventSystem.topics[topic]
						delete(chans,id)
					}else{
						delete(eventSystem.globs,id)
					}
				}
			}
		}
	}()
	log.Print("successfully started EventSystem")
}

/*
returns error when chanel is closed
*/
func safeSend(c chan interface{}, t interface{}) (err error) {
	defer func() {
		if x := recover(); x != nil {
			err = errors.New("SendError")
		}
	}()
	c <- t
	return
}

/*
Global publish function
*/
func Publish(topic string, payload interface{}) bool{
	command := &command{
		Type:    PUBLISH,
		Topic:   topic,
		Payload: payload,
		Result:  make(chan interface{}),
	}
	eventSystem.cmdChan <- command
	res := (<-command.Result).(bool)
	return res
}

/*
Global subscribe function
*/
func Subscribe(topic string) (eventChannel chan interface{},closeChannel chan bool) {
	command := &command{
		Type:    SUBSCRIBE,
		Topic:   topic,
		Result:  make(chan interface{}),
	}
	eventSystem.cmdChan <- command
	res_ := <-command.Result
	res := res_.(unsubscribeResult)
	eventChannel = res.EventChan
	closeChannel = res.CloseChan
	return
}

