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
	"path/filepath"
)

/*
Global publish function
*/
func Publish(event *Event) bool {
	command := &command{
		Type:   PUBLISH,
		Event:  event,
		Result: make(chan interface{}),
	}
	eventSystem.cmdChan <- command
	res := (<-command.Result).(bool)
	return res
}

func (eventSystem *EventSystem) publish(event *Event) bool {
	found := false
	for _, subscription := range eventSystem.globs {
		if ok, err := filepath.Match(subscription.Glob, event.Topic); ok && (err == nil) {
			if subscription.AuthLevel <= event.AuthLevel {
				subscription.EventChan <- event
				found = true
			}
		}
	}
	subscriptions := eventSystem.topics[event.Topic]
	for _, subscription := range subscriptions {
		if subscription.AuthLevel <= event.AuthLevel {
			subscription.EventChan <- event
			found = true
		}
	}
	return found
}
