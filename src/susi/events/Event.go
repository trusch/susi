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
	"fmt"
)

type Event struct {
	Id         uint64
	Topic      string
	AuthLevel  uint8
	ReturnAddr string
	Payload    interface{}
}

func NewEvent(topic string, payload interface{}) *Event {
	return &Event{
		Id:        uint64(time.Now().UnixNano()),
		Topic:     topic,
		AuthLevel: 255,
		Payload:   payload,
	}
}

func (evt *Event) String() string {
	return fmt.Sprintf("%v",*evt)
}
