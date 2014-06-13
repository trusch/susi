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
	"encoding/json"
	"flag"
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/state"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var eventQueueSize = flag.String("webstack.eventqueuesize", "100", "How many events should be queued for each session")

type eventsCmdType uint8

const (
	ADDEVENT eventsCmdType = iota
	GETEVENTS
	SUBSCRIBE
	UNSUBSCRIBE
	CLEANUP
)

type eventsCmd struct {
	Type      eventsCmdType
	Topic     string
	Id        string
	AuthLevel uint8
	Payload   interface{}
	Result    chan interface{}
}

type subscription struct {
	Topic      string
	closeChans []chan bool
}

type EventsHandler struct {
	subscriptions  map[string][]*subscription
	events         map[string][]*events.Event
	eventQueueSize int
	cmdChan        chan *eventsCmd
}

func (handler *EventsHandler) backend() {

	//Wait for cleanup and feed into handler
	go func() {
		ch, _ := events.Subscribe("session::delete", 0)
		for evt := range ch {
			handler.cmdChan <- &eventsCmd{
				Type: CLEANUP,
				Id:   strconv.Itoa(int(evt.Payload.(uint64))),
			}
		}
	}()

	for cmd := range handler.cmdChan {
		switch cmd.Type {
		case ADDEVENT:
			{
				handler.addEvent(cmd.Id, cmd.Payload.(*events.Event))
			}
		case GETEVENTS:
			{
				events := handler.getEvents(cmd.Id)
				cmd.Result <- events
			}
		case SUBSCRIBE:
			{
				handler.subscribe(cmd.Id, cmd.Topic, cmd.AuthLevel)
			}
		case UNSUBSCRIBE:
			{
				handler.unsubscribe(cmd.Id, cmd.Topic)
			}
		case CLEANUP:
			{
				subscriptions := handler.subscriptions[cmd.Id]
				for _, sub := range subscriptions {
					for _, ch := range sub.closeChans {
						ch <- true
					}
				}
				delete(handler.subscriptions, cmd.Id)
				delete(handler.events, cmd.Id)
				log.Print("cleanup of ", cmd.Id)
			}
		}
	}
}

func (handler *EventsHandler) addEvent(id string, event *events.Event) {
	events := handler.events[id]
	if len(events) >= handler.eventQueueSize {
		handler.events[id] = append(events[1:], event)
	} else {
		handler.events[id] = append(events, event)
	}
}

func (handler *EventsHandler) getEvents(id string) []*events.Event {
	events := handler.events[id]
	handler.events[id] = nil
	return events
}

func (handler *EventsHandler) subscribe(id, topic string, authlevel uint8) {
	eventChan, closeChan := events.Subscribe(topic, authlevel)
	close2 := make(chan bool)
	subscriptions := handler.subscriptions[id]
	for _, sub := range subscriptions {
		if sub.Topic == topic {
			return
		}
	}
	subscription := &subscription{
		Topic: topic,
		closeChans: []chan bool{
			closeChan, close2,
		},
	}
	handler.subscriptions[id] = append(handler.subscriptions[id], subscription)
	go func() {
		for {
			select {
			case <-close2:
				{
					return
				}
			case event := <-eventChan:
				{
					handler.cmdChan <- &eventsCmd{
						Type:    ADDEVENT,
						Id:      id,
						Payload: event,
					}
				}
			}
		}
	}()
}

func (handler *EventsHandler) unsubscribe(id, topic string) {
	subscriptions := handler.subscriptions[id]
	idx := 0
	var sub *subscription = nil
	for idx, sub = range subscriptions {
		if sub.Topic == topic {
			break
		}
	}
	if sub != nil {
		subscriptions = append(subscriptions[:idx], subscriptions[idx+1:]...)
		handler.subscriptions[id] = subscriptions
		for _, closer := range sub.closeChans {
			closer <- true
		}
	}
}

func NewEventsHandler() *EventsHandler {
	handler := new(EventsHandler)
	sizeStr := state.Get("webstack.eventqueuesize").(string)
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		log.Fatal(err)
	}
	handler.eventQueueSize = size
	handler.subscriptions = make(map[string][]*subscription)
	handler.events = make(map[string][]*events.Event)
	handler.cmdChan = make(chan *eventsCmd, 10)

	go handler.backend()

	return handler
}

func (ptr *EventsHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	cookie, _ := req.Cookie("susisession")
	id := cookie.Value
	authlevel_, _ := strconv.Atoi(req.Header.Get("authlevel"))
	authlevel := uint8(authlevel_)
	username := req.Header.Get("username")
	path := req.URL.Path
	switch {
	case strings.HasPrefix(path, "/events/publish"):
		{
			reader := io.LimitReader(req.Body, 1024)
			decoder := json.NewDecoder(reader)
			type publishMsg struct {
				Key        string      `json:"key"`
				AuthLevel  uint8       `json:"authlevel"`
				ReturnAddr string      `json:"returnaddr"`
				Payload    interface{} `json:"payload"`
			}
			msg := new(publishMsg)
			err := decoder.Decode(&msg)
			if err != nil {
				log.Print(err)
				log.Print(ioutil.ReadAll(reader))
				resp.WriteHeader(http.StatusBadRequest)
				return
			}
			if msg.AuthLevel < authlevel {
				msg.AuthLevel = authlevel
			}
			event := events.NewEvent(msg.Key, msg.Payload)
			event.AuthLevel = msg.AuthLevel
			event.ReturnAddr = msg.ReturnAddr
			event.Username = username
			events.Publish(event)
			resp.WriteHeader(http.StatusOK)
			return
		}
	case strings.HasPrefix(path, "/events/subscribe"):
		{
			reader := io.LimitReader(req.Body, 1024)
			decoder := json.NewDecoder(reader)
			type subscribeMsg struct {
				Key       string `json:"key"`
				AuthLevel uint8  `json:"authlevel"`
			}
			msg := new(subscribeMsg)
			err := decoder.Decode(&msg)
			if err != nil {
				log.Print(err)
				log.Print(ioutil.ReadAll(reader))
				resp.WriteHeader(http.StatusBadRequest)
				return
			}
			if msg.AuthLevel < authlevel {
				msg.AuthLevel = authlevel
			}
			ptr.cmdChan <- &eventsCmd{
				Type:      SUBSCRIBE,
				Id:        id,
				Topic:     msg.Key,
				AuthLevel: msg.AuthLevel,
			}
			resp.WriteHeader(http.StatusOK)
			return
		}
	case strings.HasPrefix(path, "/events/unsubscribe"):
		{
			reader := io.LimitReader(req.Body, 1024)
			decoder := json.NewDecoder(reader)
			type unsubscribeMsg struct {
				Key string `json:"key"`
			}
			msg := new(unsubscribeMsg)
			err := decoder.Decode(&msg)
			if err != nil {
				log.Print(err)
				log.Print(ioutil.ReadAll(reader))
				resp.WriteHeader(http.StatusBadRequest)
				return
			}
			ptr.cmdChan <- &eventsCmd{
				Type:  UNSUBSCRIBE,
				Id:    id,
				Topic: msg.Key,
			}
			resp.WriteHeader(http.StatusOK)
			return
		}
	case strings.HasPrefix(path, "/events/get"):
		{
			cmd := &eventsCmd{
				Type:   GETEVENTS,
				Id:     id,
				Result: make(chan interface{}),
			}
			ptr.cmdChan <- cmd
			evts := <-cmd.Result
			encoder := json.NewEncoder(resp)
			encoder.Encode(&evts)
			return
		}
	}
}
