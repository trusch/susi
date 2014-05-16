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
	"github.com/robertkrimen/otto"
	"flag"
	"log"
)

var jsRoot = flag.String("jsengine.root","./js/","where to search for backend js controllers")

type ottoCommandType uint8

const(
	SUBSCRIBE ottoCommandType = iota
	UNSUBSCRIBE
)

type ottoCommand struct {
	Type ottoCommandType
	Topic string
	Id uint64
	Result chan uint64
}

type OttoEngine struct {
	vm *otto.Otto
	input chan *ottoCommand
}




func Go(){
	ptr := new(OttoEngine)
	ptr.vm = otto.New()
	susiObj, _ := ptr.vm.Object(`({})`)
	eventsObj, _ := ptr.vm.Object(`({})`)
	
	eventsObj.Set("publish",func(call otto.FunctionCall) otto.Value {
	    log.Print(call.Argument(0).String())
		return otto.UndefinedValue()
	})

	eventsObj.Set("subscribe",func(call otto.FunctionCall) otto.Value {
	    log.Print(call.Argument(0).String())
	    return otto.UndefinedValue()
	})
	
	eventsObj.Set("unsubscribe",func(call otto.FunctionCall) otto.Value {
	    log.Print(call.Argument(0).String())
		return otto.UndefinedValue()
	})
	
	susiObj.Set("events",eventsObj)
	susiObj.Set("log",func(call otto.FunctionCall) otto.Value {
	    log.Print(call.Argument(0).String())
		return otto.UndefinedValue()
	})


	ptr.vm.Set("susi",susiObj)

	ptr.vm.Run(`susi.log("foo!")`)

}



