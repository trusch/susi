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

import (
	"time"
)

/*
Global subscribe function
*/
func Subscribe(topic string, authlevel uint8) (eventChannel chan *Event, closeChannel chan bool) {
	command := &command{
		Type: SUBSCRIBE,
		Event: &Event{
			Topic:     topic,
			AuthLevel: authlevel,
		},
		Result: make(chan interface{}),
	}
	eventSystem.cmdChan <- command
	res_ := <-command.Result
	res := res_.(subscribeResult)
	eventChannel = res.EventChan
	closeChannel = res.CloseChan
	return
}

type subscription struct {
	Topic     string
	Glob      string
	AuthLevel uint8
	EventChan chan *Event
}

type subscribeResult struct {
	EventChan chan *Event
	CloseChan chan bool
}

func (ptr *EventSystem) subscribe(topic string, authlevel uint8) (eventChannel chan *Event, closeChannel chan bool) {
	eventChannel = make(chan *Event, 10)
	closeChannel = make(chan bool)
	id := uint64(time.Now().UnixNano())
	if isGlob(topic) {
		subscription := &subscription{
			Glob:      topic,
			EventChan: eventChannel,
		}
		ptr.globs[id] = subscription
	} else {
		subscriptionsMap := ptr.topics[topic]
		if subscriptionsMap == nil {
			tmp := make(map[uint64]*subscription)
			ptr.topics[topic] = tmp
			subscriptionsMap = tmp
		}
		subscription := &subscription{
			Topic:     topic,
			EventChan: eventChannel,
			AuthLevel: authlevel,
		}
		subscriptionsMap[id] = subscription
	}
	//log.Print("subscribed to ",topic," (",id,")")
	go func() {
		<-closeChannel
		//log.Print("unsubscribed from ",topic," (",id,")")
		ptr.cmdChan <- &command{
			Type: UNSUBSCRIBE,
			Event: &Event{
				Topic:   topic,
				Payload: id,
			},
		}
	}()
	return eventChannel, closeChannel
}

func (eventSystem *EventSystem) unsubscribe(topic string, id uint64) {
	if topic != "" {
		if subscriptions, ok := eventSystem.topics[topic]; ok {
			delete(subscriptions, id)
		}
	} else {
		delete(eventSystem.globs, id)
	}
}
