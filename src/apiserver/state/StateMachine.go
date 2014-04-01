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

package state

import (
	"log"
)

const (
	SET int = iota
	GET
	PUSH
	POP
	ENQUEUE
	DEQUEUE
	UNSET
)

type command struct {
	Type   int
	Key    string
	Value  interface{}
	Return chan interface{}
}

/*
This is the global StateMaching, which handles all key-value pairs needed by the system
It provides three global functions which can be called from anywhere.
*/
type StateMachine struct {
	state      map[string]interface{}
	cmdChan    chan *command
	maxListLen int
}

var stateMachine *StateMachine

func init() {
	stateMachine = new(StateMachine)
	stateMachine.maxListLen = 32
	stateMachine.cmdChan = make(chan *command, 10)
	stateMachine.state = make(map[string]interface{})
	go func() {
		for cmd := range stateMachine.cmdChan {
			switch cmd.Type {
			case SET:
				{
					stateMachine.state[cmd.Key] = cmd.Value
				}
			case GET:
				{
					cmd.Return <- stateMachine.state[cmd.Key]
				}
			case PUSH,ENQUEUE:
				{
					old, ok := stateMachine.state[cmd.Key]
					if !ok {
						stateMachine.state[cmd.Key] = []interface{}{cmd.Value}
					} else if arr,ok := old.([]interface{}); ok {
						arr = append(arr, cmd.Value)
						if stateMachine.maxListLen != 0 && len(arr) > stateMachine.maxListLen {
							arr = arr[1:]
						}
						stateMachine.state[cmd.Key] = arr
					} else {
						arr := []interface{}{old,cmd.Value}
						stateMachine.state[cmd.Key] = arr
					}
				}
			case POP:
				{
					if arr_,ok := stateMachine.state[cmd.Key]; ok {
						if arr,ok := arr_.([]interface{}); ok && len(arr)>=1 {
							val := arr[len(arr)-1]
							arr = arr[:len(arr)-1]
							stateMachine.state[cmd.Key] = arr
							cmd.Return <- val
						}else{
							cmd.Return <- arr_
						}
					}else{
						cmd.Return <- nil
					}
				}
			case DEQUEUE:
				{
					if arr_,ok := stateMachine.state[cmd.Key]; ok {
						if arr,ok := arr_.([]interface{}); ok && len(arr)>=1{
							val := arr[0]
							arr = arr[1:]
							stateMachine.state[cmd.Key] = arr
							cmd.Return <- val
						}else{
							cmd.Return <- arr_
						}
					}else{
						cmd.Return <- nil
					}
				}
			case UNSET:
				{
					delete(stateMachine.state,cmd.Key)
				}
			}
		}
	}()
	log.Print("successfully started StateMachine")
}

/*
This sets a global variable
*/
func Set(key string, val interface{}) {
	stateMachine.cmdChan <- &command{
		Type:  SET,
		Key:   key,
		Value: val,
	}
}

/*
This returns a global variable
*/
func Get(key string) interface{} {
	cmd := &command{
		Type:   GET,
		Key:    key,
		Return: make(chan interface{}),
	}
	stateMachine.cmdChan <- cmd
	return <-cmd.Return
}

func Unset(key string) {
	stateMachine.cmdChan <- &command{
		Type:  UNSET,
		Key:   key,
	}
}

func Enqueue(key string, val interface{}) {
	stateMachine.cmdChan <- &command{
		Type:  ENQUEUE,
		Key:   key,
		Value: val,
	}
}

func Push(key string, val interface{}) {
	stateMachine.cmdChan <- &command{
		Type:  PUSH,
		Key:   key,
		Value: val,
	}
}

func Dequeue(key string) interface{} {
	cmd := &command{
		Type:   DEQUEUE,
		Key:    key,
		Return: make(chan interface{}),
	}
	stateMachine.cmdChan <- cmd
	return <-cmd.Return
}

func Pop(key string) interface{} {
	cmd := &command{
		Type:   POP,
		Key:    key,
		Return: make(chan interface{}),
	}
	stateMachine.cmdChan <- cmd
	return <-cmd.Return
}
