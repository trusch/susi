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
	"flag"
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/state"
	"log"
	"strconv"
	"time"
)

var sessionLifetime = flag.String("webstack.session.lifetime", "1800", "how many seconds should a session stay alive before being invalidated")
var sessionCheckInterval = flag.String("webstack.session.checkinterval", "10", "check interval in seconds")

type Session struct {
	Id         uint64 `json:"-"`
	User       string `json:"user"`
	AuthLevel  int    `json:"authlevel"`
	ValidUntil int64  `json:"validuntil"`
}

type sessionCommandType uint8

const (
	ADDSESSION sessionCommandType = iota
	DELSESSION
	UPDATESESSION
	GETSESSION
)

type sessionCommand struct {
	Type      sessionCommandType
	Username  string
	AuthLevel int
	Id        uint64
	Return    chan interface{}
}

type SessionManager struct {
	sessions []*Session
	commands chan sessionCommand
}

func (ptr *SessionManager) addSession(user string, authlevel int) (id uint64) {
	id = uint64(time.Now().UnixNano())
	lifetimeStr := state.Get("webstack.session.lifetime").(string)
	lifetime, err := strconv.ParseInt(lifetimeStr, 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	session := &Session{
		Id:         id,
		User:       user,
		AuthLevel:  authlevel,
		ValidUntil: time.Now().Unix() + lifetime,
	}
	ptr.sessions = append(ptr.sessions, session)
	return id
}

func (ptr *SessionManager) delSession(id uint64) {
	for idx, session := range ptr.sessions {
		if session.Id == id {
			ptr.sessions = append(ptr.sessions[:idx], ptr.sessions[idx+1:]...)
			event := events.NewEvent("session::delete", id)
			event.AuthLevel = 0
			events.Publish(event)
			break
		}
	}
}

func (ptr *SessionManager) updateSession(id uint64) {
	lifetimeStr := state.Get("webstack.session.lifetime").(string)
	lifetime, err := strconv.ParseInt(lifetimeStr, 10, 64)
	if err != nil {
		log.Fatal(err)
	}
	for _, session := range ptr.sessions {
		if session.Id == id {
			session.ValidUntil = time.Now().Unix() + lifetime
			break
		}
	}
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
			event := events.NewEvent("session::delete", session.Id)
			event.AuthLevel = 0
			events.Publish(event)
		}
	}
	ptr.sessions = newSessions
}

func (ptr *SessionManager) backend() {
	intervalStr := state.Get("webstack.session.checkinterval").(string)
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
						cmd.Return <- ptr.addSession(cmd.Username, cmd.AuthLevel)
					}
				case DELSESSION:
					{
						ptr.delSession(cmd.Id)
					}
				case UPDATESESSION:
					{
						ptr.updateSession(cmd.Id)
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

func (ptr *SessionManager) AddSession(name string, authlevel int) uint64 {
	ret := make(chan interface{})
	ptr.commands <- sessionCommand{
		Type:      ADDSESSION,
		Username:  name,
		AuthLevel: authlevel,
		Return:    ret,
	}
	return (<-ret).(uint64)
}

func (ptr *SessionManager) DelSession(id uint64) {
	ptr.commands <- sessionCommand{
		Type: DELSESSION,
		Id:   id,
	}
}

func (ptr *SessionManager) UpdateSession(id uint64) {
	ptr.commands <- sessionCommand{
		Type: UPDATESESSION,
		Id:   id,
	}
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
