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

func Awnser(requestEvent *Event, data interface{}) {
	if requestEvent.ReturnAddr != "" {
		event := NewEvent(requestEvent.ReturnAddr, map[string]interface{}{
			"error": false,
			"data":  data,
		})
		event.AuthLevel = requestEvent.AuthLevel
		Publish(event)
	}
}

func AwnserError(requestEvent *Event, message string) {
	if requestEvent.ReturnAddr != "" {
		event := NewEvent(requestEvent.ReturnAddr, map[string]interface{}{
			"error": true,
			"data":  message,
		})
		event.AuthLevel = requestEvent.AuthLevel
		Publish(event)
	}
}
