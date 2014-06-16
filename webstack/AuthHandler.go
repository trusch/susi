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
	"crypto/aes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"flag"
	"github.com/trusch/susi/authentification"
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/session"
	"github.com/trusch/susi/state"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var cookieKey = flag.String("webstack.cookiekey", "foobar", "The key which is used to encrypt the cookies")

type AuthHandler struct {
	defaultHandler http.Handler
	cookieKey      []byte
}

func NewAuthHandler(defaultHandler http.Handler) *AuthHandler {
	result := new(AuthHandler)
	result.defaultHandler = defaultHandler

	cookieKeyStr := state.Get("webstack.cookiekey").(string)
	hash := sha512.New()
	hash.Write([]byte(cookieKeyStr))
	result.cookieKey = hash.Sum(result.cookieKey)[:32]

	return result
}

func (ptr *AuthHandler) addSession(resp http.ResponseWriter) (uint64, error) {
	data, err := events.Request("session::add", map[string]interface{}{
		"username":  "anonymous",
		"authlevel": uint8(3),
	})
	if err != nil {
		return 0, err
	}
	sessionId := data.(uint64)
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
	if err != nil {
		log.Print(err)
		return ptr.addSession(resp)
	}
	_, err = events.Request("session::get", sessionId)
	if err != nil {
		log.Printf("dont find session... (%v)", err)
		return ptr.addSession(resp)
	}
	return sessionId, nil
}

func (ptr *AuthHandler) checkUser(username, password string) *authentification.User {
	data, err := events.Request("authentification::checkuser", map[string]interface{}{
		"username": username,
		"password": password,
	})
	if err != nil {
		return nil
	}
	return data.(*authentification.User)
}

func (ptr *AuthHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	sessionId, err := ptr.sessionHandling(resp, req)
	data, err := events.Request("session::get", sessionId)
	if err != nil {
		log.Fatal(err)
	}
	session := data.(*session.Session)
	req.Header.Del("authlevel")
	req.Header.Add("authlevel", strconv.Itoa(int(session.Data["authlevel"].(uint8))))
	req.Header.Del("username")
	req.Header.Add("username", session.Data["username"].(string))
	req.Header.Del("sessionid")
	req.Header.Add("sessionid", strconv.Itoa(int(session.Id)))
	//log.Print("SESSION:", session)
	path := req.URL.Path
	if strings.HasPrefix(path, "/auth") {
		switch {
		case strings.HasPrefix(path, "/auth/login"):
			{
				username := ""
				password := ""
				reader := io.LimitReader(req.Body, 1024)
				decoder := json.NewDecoder(reader)
				type authMsg struct {
					Username string `json:"username"`
					Password string `json:"password"`
				}
				msg := new(authMsg)
				err = decoder.Decode(&msg)
				username = msg.Username
				password = msg.Password
				if err != nil {
					log.Print(err)
					vals := req.URL.Query()
					username = vals.Get("username")
					password = vals.Get("password")
				}
				if user := ptr.checkUser(username, password); user != nil {
					session.Data["authlevel"] = user.AuthLevel
					session.Data["username"] = user.Username
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
				session.Data["authlevel"] = uint8(3)
				session.Data["username"] = "anonymous"
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
				events.Request("session::touch", sessionId)
				resp.WriteHeader(http.StatusOK)
				return
			}
		}
	}

	ptr.defaultHandler.ServeHTTP(resp, req)
}
