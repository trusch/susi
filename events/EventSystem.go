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
	topics  map[string]map[uint64]*subscription
	globs   map[uint64]*subscription
}

type globChan struct {
	EventChan chan interface{}
	Glob      string
}

func isGlob(pattern string) bool {
	return strings.IndexAny(pattern, "*?[") >= 0
}

type command struct {
	Type   CommandType
	Event  *Event
	Result chan interface{}
}

var eventSystem *EventSystem

func Go() {
	eventSystem = new(EventSystem)
	eventSystem.cmdChan = make(chan *command, 10)
	eventSystem.topics = make(map[string]map[uint64]*subscription)
	eventSystem.globs = make(map[uint64]*subscription)
	go func() {
		for cmd := range eventSystem.cmdChan {
			switch cmd.Type {
			case SUBSCRIBE:
				{
					ec, cc := eventSystem.subscribe(cmd.Event.Topic, cmd.Event.AuthLevel)
					cmd.Result <- subscribeResult{
						EventChan: ec,
						CloseChan: cc,
					}
				}
			case PUBLISH:
				{
					cmd.Result <- eventSystem.publish(cmd.Event)
				}
			case UNSUBSCRIBE:
				{
					eventSystem.unsubscribe(cmd.Event.Topic, cmd.Event.Payload.(uint64))
				}
			}
		}
	}()
	log.Print("successfully started EventSystem")
}

func Reset() {
	for topic, subs := range eventSystem.topics {
		for id, _ := range subs {
			eventSystem.unsubscribe(topic, id)
		}
	}
	for id, _ := range eventSystem.globs {
		eventSystem.unsubscribe("", id)
	}
}
