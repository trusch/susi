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

package session

import (
	"flag"
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/state"
	"log"
	"strconv"
	"time"
)

var sessionLifetime = flag.String("session.lifetime", "1800", "how many seconds should a session stay alive before being invalidated")
var sessionCheckInterval = flag.String("session.checkinterval", "10", "check interval in seconds")

type Session struct {
	Id         uint64                 `json:"-"`
	ValidUntil int64                  `json:"validuntil"`
	Data       map[string]interface{} `json:"data"`
}

type sessionCommandType uint8

const (
	ADDSESSION sessionCommandType = iota
	DELSESSION
	TOUCHSESSION
	UPDATESESSION
	GETSESSION
)

type sessionCommand struct {
	Type   sessionCommandType
	Data   map[string]interface{}
	Id     uint64
	Return chan interface{}
}

type SessionManager struct {
	sessions []*Session
	commands chan sessionCommand
}

func (ptr *SessionManager) addSession(data map[string]interface{}) (id uint64) {
	id = uint64(time.Now().UnixNano())
	lifetimeStr := state.Get("session.lifetime").(string)
	lifetime, err := strconv.ParseInt(lifetimeStr, 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	session := &Session{
		Id:         id,
		Data:       data,
		ValidUntil: time.Now().Unix() + lifetime,
	}
	ptr.sessions = append(ptr.sessions, session)
	return id
}

func (ptr *SessionManager) delSession(id uint64) bool {
	for idx, session := range ptr.sessions {
		if session.Id == id {
			ptr.sessions = append(ptr.sessions[:idx], ptr.sessions[idx+1:]...)
			event := events.NewEvent("session::deleted", id)
			event.AuthLevel = 0
			events.Publish(event)
			return true
		}
	}
	return false
}

func (ptr *SessionManager) touchSession(id uint64) bool {
	lifetimeStr := state.Get("session.lifetime").(string)
	lifetime, err := strconv.ParseInt(lifetimeStr, 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	for _, session := range ptr.sessions {
		if session.Id == id {
			session.ValidUntil = time.Now().Unix() + lifetime
			return true
		}
	}
	return false
}

func (ptr *SessionManager) getSession(id uint64) *Session {
	for _, session := range ptr.sessions {
		if session.Id == id {
			return session
		}
	}
	return nil
}

func (ptr *SessionManager) checkSessions() {
	newSessions := make([]*Session, 0, len(ptr.sessions))
	now := time.Now().Unix()
	for _, session := range ptr.sessions {
		if session.ValidUntil > now {
			newSessions = append(newSessions, session)
		} else {
			event := events.NewEvent("session::deleted", session.Id)
			event.AuthLevel = 0
			events.Publish(event)
		}
	}
	ptr.sessions = newSessions
}

func (ptr *SessionManager) backend() {
	intervalStr := state.Get("session.checkinterval").(string)
	interval, err := strconv.ParseInt(intervalStr, 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	ticker := time.Tick(time.Duration(interval) * time.Second)
	for {
		select {
		case cmd := <-ptr.commands:
			{
				switch cmd.Type {
				case ADDSESSION:
					{
						cmd.Return <- ptr.addSession(cmd.Data)
					}
				case DELSESSION:
					{
						cmd.Return <- ptr.delSession(cmd.Id)
					}
				case TOUCHSESSION:
					{
						cmd.Return <- ptr.touchSession(cmd.Id)
					}
				case GETSESSION:
					{
						cmd.Return <- ptr.getSession(cmd.Id)
					}
				}
			}
		case <-ticker:
			{
				ptr.checkSessions()
			}
		}
	}
}

func (ptr *SessionManager) AddSession(data map[string]interface{}) uint64 {
	ret := make(chan interface{})
	ptr.commands <- sessionCommand{
		Type:   ADDSESSION,
		Data:   data,
		Return: ret,
	}
	return (<-ret).(uint64)
}

func (ptr *SessionManager) DelSession(id uint64) bool {
	ret := make(chan interface{})
	ptr.commands <- sessionCommand{
		Type:   DELSESSION,
		Id:     id,
		Return: ret,
	}
	return (<-ret).(bool)
}

func (ptr *SessionManager) TouchSession(id uint64) bool {
	ret := make(chan interface{})
	ptr.commands <- sessionCommand{
		Type:   TOUCHSESSION,
		Id:     id,
		Return: ret,
	}
	log.Print("finished touch")
	return (<-ret).(bool)
}

func (ptr *SessionManager) GetSession(id uint64) *Session {
	ret := make(chan interface{})
	ptr.commands <- sessionCommand{
		Type:   GETSESSION,
		Id:     id,
		Return: ret,
	}
	return (<-ret).(*Session)
}

func NewSessionManager() *SessionManager {
	manager := new(SessionManager)
	manager.commands = make(chan sessionCommand, 10)
	manager.sessions = make([]*Session, 0, 32)
	go manager.backend()
	return manager
}

var sessionManager *SessionManager

func Go() {

	addSessionChan, _ := events.Subscribe("session::add", 0)
	delSessionChan, _ := events.Subscribe("session::del", 0)
	getSessionChan, _ := events.Subscribe("session::get", 0)
	touchSessionChan, _ := events.Subscribe("session::touch", 0)

	sessionManager = NewSessionManager()

	go func() {
		for {
			select {
			case event := <-addSessionChan:
				{
					if event == nil {
						return
					}
					if event.AuthLevel > 0 {
						events.AwnserError(event, "need authlevel zero")
						break
					}
					var id uint64 = 0
					if data, ok := event.Payload.(map[string]interface{}); ok {
						id = sessionManager.AddSession(data)
					} else {
						id = sessionManager.AddSession(nil)
					}
					if id == 0 {
						events.AwnserError(event, "error while adding session")
					} else {
						events.Awnser(event, id)
					}
				}
			case event := <-delSessionChan:
				{
					if event == nil {
						return
					}
					if event.AuthLevel > 0 {
						events.AwnserError(event, "need authlevel zero")
						break
					}
					success := false
					if id, ok := event.Payload.(uint64); ok {
						success = sessionManager.DelSession(id)
					}
					if success {
						events.Awnser(event, nil)
					} else {
						events.AwnserError(event, "error deleting session")
					}
				}
			case event := <-touchSessionChan:
				{
					if event == nil {
						return
					}
					if event.AuthLevel > 0 {
						events.AwnserError(event, "need authlevel zero")
						break
					}
					success := false
					if id, ok := event.Payload.(uint64); ok {
						success = sessionManager.TouchSession(id)
					}
					if success {
						events.Awnser(event, nil)
					} else {
						events.AwnserError(event, "error touching session")
					}
					log.Print("finished touch")
				}
			case event := <-getSessionChan:
				{
					if event == nil {
						return
					}
					if event.AuthLevel > 0 {
						events.AwnserError(event, "need authlevel zero")
						break
					}
					success := false
					var session *Session = nil
					if id, ok := event.Payload.(uint64); ok {
						session = sessionManager.GetSession(id)
						success = (session != nil)
					}
					if success {
						events.Awnser(event, session)
					} else {
						events.AwnserError(event, "error getting session")
					}
				}
			}
		}
	}()
}
