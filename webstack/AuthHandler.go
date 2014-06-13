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
	"code.google.com/p/go-uuid/uuid"
	"crypto/aes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"flag"
	"github.com/trusch/susi/authentification"
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/state"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var cookieKey = flag.String("webstack.cookiekey", "foobar", "The key which is used to encrypt the cookies")

type AuthHandler struct {
	defaultHandler http.Handler
	sessionManager *SessionManager
	cookieKey      []byte
}

func NewAuthHandler(defaultHandler http.Handler) *AuthHandler {
	result := new(AuthHandler)
	result.defaultHandler = defaultHandler
	result.sessionManager = NewSessionManager()

	cookieKeyStr := state.Get("webstack.cookiekey").(string)
	hash := sha512.New()
	hash.Write([]byte(cookieKeyStr))
	result.cookieKey = hash.Sum(result.cookieKey)[:32]

	return result
}

func (ptr *AuthHandler) addSession(resp http.ResponseWriter) (uint64, error) {
	sessionId := ptr.sessionManager.AddSession("anonymous", 3)
	sessionIdBytes := []byte(strconv.FormatUint(sessionId, 16))
	cipher, err := aes.NewCipher(ptr.cookieKey)
	if err != nil {
		return 0, err
	}
	cipher.Encrypt(sessionIdBytes, sessionIdBytes)
	cookieStr := base64.StdEncoding.EncodeToString(sessionIdBytes)
	cookie := &http.Cookie{Name: "susisession", Value: cookieStr, Path: "/"}
	http.SetCookie(resp, cookie)
	return sessionId, nil
}

func (ptr *AuthHandler) getSession(req *http.Request) (uint64, error) {
	cookie, err := req.Cookie("susisession")
	if err != nil {
		return 0, err
	}
	sessionIdBytes, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return 0, err
	}
	cipher, err := aes.NewCipher(ptr.cookieKey)
	if err != nil {
		return 0, err
	}
	cipher.Decrypt(sessionIdBytes, sessionIdBytes)
	sessionId, err := strconv.ParseUint(string(sessionIdBytes), 16, 64)
	if err != nil {
		return 0, err
	}
	cookie.Value = string(sessionIdBytes)
	return sessionId, nil
}

func (ptr *AuthHandler) sessionHandling(resp http.ResponseWriter, req *http.Request) (uint64, error) {
	sessionId, err := ptr.getSession(req)
	//log.Println(sessionId)
	if err != nil {
		log.Print(err)
		return ptr.addSession(resp)
	}
	session := ptr.sessionManager.GetSession(sessionId)
	if session == nil {
		log.Print("dont find session...")
		return ptr.addSession(resp)
	}
	return sessionId, nil
}

func (ptr *AuthHandler) checkUser(username, password string) *authentification.User {
	awnserTopic := uuid.New()
	awnserChan, closeChan := events.Subscribe(awnserTopic, 0)
	event := events.NewEvent("authentification::checkuser", map[string]interface{}{
		"username": username,
		"password": password,
	})
	event.ReturnAddr = awnserTopic
	event.AuthLevel = 0
	events.Publish(event)
	awnser_ := <-awnserChan
	closeChan <- true
	awnser := awnser_.Payload.(*authentification.AwnserData)
	if awnser.Success {
		return awnser.Message.(*authentification.User)
	}
	return nil
}

func (ptr *AuthHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	sessionId, err := ptr.sessionHandling(resp, req)
	session := ptr.sessionManager.GetSession(sessionId)
	req.Header.Del("authlevel")
	req.Header.Add("authlevel", strconv.Itoa(int(session.AuthLevel)))
	req.Header.Del("username")
	req.Header.Add("username", session.User)
	//log.Print("SESSION:", session)
	path := req.URL.Path
	if strings.HasPrefix(path, "/auth") {
		switch {
		case strings.HasPrefix(path, "/auth/login"):
			{
				reader := io.LimitReader(req.Body, 1024)
				decoder := json.NewDecoder(reader)
				type authMsg struct {
					Username string `json:"username"`
					Password string `json:"password"`
				}
				msg := new(authMsg)
				err = decoder.Decode(&msg)
				if err != nil {
					log.Print(err)
					log.Print(ioutil.ReadAll(reader))
					resp.WriteHeader(http.StatusBadRequest)
					return
				}
				if user := ptr.checkUser(msg.Username, msg.Password); user != nil {
					session.AuthLevel = user.AuthLevel
					session.User = user.Username
					log.Print("successfully logged in for user: ", msg.Username)
					resp.WriteHeader(http.StatusOK)
					return
				} else {
					log.Print("unauthorized login request for user: ", msg.Username)
					resp.WriteHeader(http.StatusUnauthorized)
					return
				}
			}
		case strings.HasPrefix(path, "/auth/logout"):
			{
				session.AuthLevel = 3
				session.User = "anonymous"
				resp.WriteHeader(http.StatusOK)
				return
			}
		case strings.HasPrefix(path, "/auth/info"):
			{
				encoder := json.NewEncoder(resp)
				encoder.Encode(session)
				return
			}
		case strings.HasPrefix(path, "/auth/keepalive"):
			{
				ptr.sessionManager.UpdateSession(sessionId)
				resp.WriteHeader(http.StatusOK)
				return
			}
		}
	}

	ptr.defaultHandler.ServeHTTP(resp, req)
}
