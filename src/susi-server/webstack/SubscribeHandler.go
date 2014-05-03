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

package webstack

import (
	"../events"
	//"../state"
	"flag"
	"net/http"
)

var eventQueueSize = flag.String("webstack.eventqueuesize","100","How many events should be queued for each session")

type SubscribeHandler struct {
	eventsToDeliver map[string][]*events.Event
}

func (ptr *SubscribeHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request){

}
