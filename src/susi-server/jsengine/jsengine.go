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

package jsengine

import (
	"../events"
	"../state"
	"flag"
	"github.com/robertkrimen/otto"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var jsRoot = flag.String("jsengine.root", "./controller/js/", "where to search for backend js controllers")

func isGlob(pattern string) bool {
	return strings.IndexAny(pattern, "*?[") >= 0
}

type subscription struct {
	CloseChan chan bool
	Functions map[int64]*otto.FunctionCall
}

type OttoEngine struct {
	vm            *otto.Otto
	input         chan *events.Event
	subscriptions map[string]*subscription
}

func (ptr *OttoEngine) dispatchEvent(event *events.Event) {
	for key, subscription := range ptr.subscriptions {
		var match bool = false
		if isGlob(key) {
			ok, err := filepath.Match(key, event.Topic)
			if err != nil {
				log.Print(err)
				return
			}
			match = ok
		}
		if !match {
			match = (key == event.Topic)
		}
		if match {
			for _, functionCall := range subscription.Functions {
				eventVal, _ := ptr.vm.ToValue(event)
				ptr.vm.Call("Function.call.call", nil, functionCall.Argument(1), nil, eventVal)
			}
		}
	}
}

func (ptr *OttoEngine) subscribe(topic string, function *otto.FunctionCall, authlevel uint8) int64 {
	sub, ok := ptr.subscriptions[topic]
	if !ok {
		sub = new(subscription)
		sub.Functions = make(map[int64]*otto.FunctionCall)
		dataChan, closeChan := events.Subscribe(topic, authlevel)
		sub.CloseChan = closeChan
		go func() {
			for event := range dataChan {
				ptr.input <- event
			}
		}()
		ptr.subscriptions[topic] = sub
	}
	id := time.Now().UnixNano()
	sub.Functions[id] = function
	return id
}

func (ptr *OttoEngine) loadJS(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Print(err)
		return
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Print("JS Error: ", err)
		return
	}
	_, err = ptr.vm.Run(string(data))
	if err != nil {
		log.Print("JS Error: ", err)
		return
	}
	log.Print("Successfully loaded js file: ", filename)
}

func (ptr *OttoEngine) searchForJS(root string) {
	if d, err := os.Open(root); err == nil {
		d.Close()
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			name := info.Name()
			if !info.IsDir() && (strings.HasSuffix(name, "js")) {
				ptr.loadJS(path)
			}
			return nil
		})
	}
}

func Go() {
	ptr := new(OttoEngine)
	ptr.vm = otto.New()
	ptr.input = make(chan *events.Event, 10)
	ptr.subscriptions = make(map[string]*subscription)

	go func() {
		for event := range ptr.input {
			ptr.dispatchEvent(event)
		}
	}()

	susiObj, _ := ptr.vm.Object(`({})`)
	eventsObj, _ := ptr.vm.Object(`({})`)

	eventsObj.Set("publish", func(call otto.FunctionCall) otto.Value {

		keyVal := call.Argument(0)
		dataVal := call.Argument(1)
		authlevelVal := call.Argument(2)
		returnaddrVal := call.Argument(3)

		if !keyVal.IsString() {
			return otto.FalseValue()
		}

		key, err1 := keyVal.ToString()
		authlevel, err2 := authlevelVal.ToInteger()
		returnaddr, err3 := returnaddrVal.ToString()
		data, err4 := dataVal.Export()

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			return otto.FalseValue()
		}

		event := events.NewEvent(key, data)
		event.AuthLevel = uint8(authlevel)
		event.ReturnAddr = returnaddr

		events.Publish(event)

		return otto.TrueValue()
	})

	eventsObj.Set("subscribe", func(call otto.FunctionCall) otto.Value {
		keyVal := call.Argument(0)
		authlevelVal := call.Argument(2)
		authlevel, err := authlevelVal.ToInteger()
		key, err1 := keyVal.ToString()

		if !keyVal.IsString() || err != nil || err1 != nil {
			return otto.FalseValue()
		}

		id := ptr.subscribe(key, &call, uint8(authlevel))

		idVal, _ := otto.ToValue(id)
		return idVal
	})

	eventsObj.Set("unsubscribe", func(call otto.FunctionCall) otto.Value {
		keyVal := call.Argument(0)
		idVal := call.Argument(1)
		id, err := idVal.ToInteger()
		key, err1 := keyVal.ToString()

		if err != nil || err1 != nil {
			return otto.FalseValue()
		}

		subscription := ptr.subscriptions[key]
		delete(subscription.Functions, id)
		if len(subscription.Functions) == 0 {
			subscription.CloseChan <- true
			delete(ptr.subscriptions, key)
		}
		return otto.TrueValue()
	})

	susiObj.Set("events", eventsObj)
	susiObj.Set("log", func(call otto.FunctionCall) otto.Value {
		log.Print(call.Argument(0).String())
		return otto.UndefinedValue()
	})

	ptr.vm.Set("susi", susiObj)

	jsDir := state.Get("jsengine.root").(string)
	ptr.searchForJS(jsDir)

	log.Print("Successfully started otto JS engine")
}
