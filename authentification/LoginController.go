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

package authentification

import (
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/session"
	"log"
)

func GoLoginController() {
	loginChan, _ := events.Subscribe("controller::auth::login", 0)
	logoutChan, _ := events.Subscribe("controller::auth::logout", 0)
	infoChan, _ := events.Subscribe("controller::auth::info", 0)

	go func() {
		for {
			select {
			case event := <-loginChan:
				{
					log.Print("got login request")
					payload, ok := event.Payload.(map[string]interface{})
					if !ok {
						events.AwnserError(event, "malformed payload")
						break
					}
					username, ok1 := payload["username"].(string)
					password, ok2 := payload["password"].(string)
					if !ok1 || !ok2 {
						events.AwnserError(event, "malformed payload")
						break
					}
					user, err := events.Request("authentification::checkuser", map[string]interface{}{
						"username": username,
						"password": password,
					})
					if err == nil {
						sessionData, err := events.Request("session::get", event.SessionId)
						if err != nil {
							log.Print(err)
							events.AwnserError(event, err.Error())
							break
						}
						session := sessionData.(*session.Session)
						session.Data["username"] = username
						session.Data["authlevel"] = uint8(user.(*User).AuthLevel)
						events.Awnser(event, map[string]interface{}{
							"username": username,
						})
					} else {
						events.AwnserError(event, err.Error())
					}
					break
				}
			case event := <-logoutChan:
				{
					sessionData, err := events.Request("session::get", event.SessionId)
					if err != nil {
						log.Print(err)
						events.AwnserError(event, err.Error())
						break
					}
					sessionData.(*session.Session).Data["username"] = "anonymous"
					sessionData.(*session.Session).Data["authlevel"] = uint8(3)
					events.Awnser(event, "successfully logged out")
					break
				}
			case event := <-infoChan:
				{
					sessionData, err := events.Request("session::get", event.SessionId)
					if err != nil {
						log.Print(err)
						events.AwnserError(event, err.Error())
						break
					}
					session := sessionData.(*session.Session)
					events.Awnser(event, map[string]interface{}{
						"username":  session.Data["username"],
						"authlevel": session.Data["authlevel"],
					})
					break
				}
			}
		}
	}()
}
