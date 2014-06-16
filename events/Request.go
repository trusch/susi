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
	"errors"
	"strconv"
	"time"
)

func Request(topic string, payload interface{}) (interface{}, error) {
	awnserTopic := "result" + strconv.Itoa(int(time.Now().UnixNano()))
	awnserChan, closeChan := Subscribe(awnserTopic, 0)
	defer func() { closeChan <- true }()
	event := NewEvent(topic, payload)
	event.AuthLevel = 0
	event.ReturnAddr = awnserTopic
	Publish(event)
	awnserEvent := <-awnserChan
	if dataMap, ok := awnserEvent.Payload.(map[string]interface{}); ok {
		err, ok1 := dataMap["error"].(bool)
		data, ok2 := dataMap["data"]
		if ok1 && ok2 {
			if err {
				if errMessage, ok := data.(string); ok {
					return nil, errors.New(errMessage)
				}
			} else {
				return data, nil
			}
		}
	}
	return nil, errors.New("malformed awnser packet")
}
