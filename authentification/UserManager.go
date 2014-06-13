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
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/state"
	"log"
	"os"
	"strconv"
)

var hashRounds = flag.String("authentification.hashRounds", "64", "How many hash rounds to perform")
var usersFile = flag.String("authentification.usersFile", "users.json", "The file where the login data will be saved")

func NewUserManager() *UserManager {
	ptr := new(UserManager)
	ptr.cmds = make(chan userManagerCommand, 10)
	ptr.users = make([]*User, 0, 32)

	ptr.usersFile = state.Get("authentification.usersFile").(string)
	roundsStr := state.Get("authentification.hashRounds").(string)
	if rounds, err := strconv.ParseInt(roundsStr, 10, 64); err != nil {
		log.Print(err)
		ptr.hashRounds = 32
	} else {
		ptr.hashRounds = int(rounds)
	}

	ptr.Load()
	go ptr.backend()

	return ptr
}
func (ptr *UserManager) AddUser(name, password string, authlevel uint8) bool {
	cmd := userManagerCommand{
		Type:   ADDUSER,
		Return: make(chan interface{}),
		User: &User{
			Username:  name,
			Password:  password,
			AuthLevel: authlevel,
		},
	}
	ptr.cmds <- cmd
	return (<-cmd.Return).(bool)
}

func (ptr *UserManager) DelUser(name string) bool {
	cmd := userManagerCommand{
		Type:   DELUSER,
		Return: make(chan interface{}),
		User: &User{
			Username: name,
		},
	}
	ptr.cmds <- cmd
	return (<-cmd.Return).(bool)
}

func (ptr *UserManager) CheckUser(name, password string) *User {
	cmd := userManagerCommand{
		Type:   CHECKUSER,
		Return: make(chan interface{}),
		User: &User{
			Username: name,
			Password: password,
		},
	}
	ptr.cmds <- cmd
	ret := <-cmd.Return
	if ret == nil {
		var r *User = nil
		return r
	}
	return (ret.(*User))
}

type User struct {
	ID        uint64
	Username  string
	Password  string
	AuthLevel uint8
}

func (user *User) HashPassword(rounds int) {
	for i := 0; i < rounds; i++ {
		buff := &bytes.Buffer{}
		hash := sha512.New()
		encoder := base64.NewEncoder(base64.StdEncoding, buff)
		hash.Write([]byte(user.Password))
		encoder.Write(hash.Sum(make([]byte, 0, hash.Size())))
		user.Password = buff.String()
	}
}

func (user *User) String() string {
	return fmt.Sprintf("%v", *user)
}

type userManagerCommandType int

const (
	ADDUSER userManagerCommandType = iota
	DELUSER
	CHECKUSER
)

type userManagerCommand struct {
	User   *User
	Type   userManagerCommandType
	Return chan interface{}
}

type UserManager struct {
	cmds       chan userManagerCommand
	users      []*User
	hashRounds int
	usersFile  string
}

func (manager *UserManager) backend() {
MAINLOOP:
	for cmd := range manager.cmds {
		switch cmd.Type {
		case ADDUSER:
			{
				maxID := uint64(0)
				for _, user := range manager.users {
					if user.Username == cmd.User.Username {
						cmd.Return <- false
						continue MAINLOOP
					}
					if user.ID > maxID {
						maxID = user.ID
					}
				}
				cmd.User.HashPassword(manager.hashRounds)
				cmd.User.ID = maxID + 1
				manager.users = append(manager.users, cmd.User)
				manager.Save()
				cmd.Return <- true
			}
		case DELUSER:
			{
				for idx, user := range manager.users {
					if user.Username == cmd.User.Username {
						manager.users = append(manager.users[:idx], manager.users[idx+1:]...)
						manager.Save()
						cmd.Return <- true
						continue MAINLOOP
					}
				}
				cmd.Return <- false
			}
		case CHECKUSER:
			{
				cmd.User.HashPassword(manager.hashRounds)
				for _, user := range manager.users {
					if user.Username == cmd.User.Username {
						if user.Password == cmd.User.Password {
							cmd.User.Password = ""
							cmd.User.ID = user.ID
							cmd.User.AuthLevel = user.AuthLevel
							cmd.Return <- cmd.User
							continue MAINLOOP
						} else {
							log.Printf("%v %v", user.Password, cmd.User.Password)
							cmd.Return <- nil
							continue MAINLOOP
						}
					}
				}
				cmd.Return <- nil
			}
		}
	}
}

func (ptr *UserManager) Load() {
	f, err := os.Open(ptr.usersFile)
	if err != nil {
		ptr.users = make([]*User, 0)
		log.Print(err)
		return
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&ptr.users)
	if err != nil {
		log.Print(err)
		return
	}
	log.Print("loaded users: ", ptr.users)
}

func (ptr *UserManager) Save() {
	f, err := os.Create(ptr.usersFile)
	if err != nil {
		log.Print(err)
		return
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.Encode(ptr.users)
}

type AwnserData struct {
	Success bool        `json:"success"`
	Message interface{} `json:"message,omitempty"`
}

var userManagerRef *UserManager

func Go() {
	userManager := NewUserManager()
	userManagerRef = userManager
	addUserChan, _ := events.Subscribe("authentification::adduser", 0)
	delUserChan, _ := events.Subscribe("authentification::deluser", 0)
	checkUserChan, _ := events.Subscribe("authentification::checkuser", 0)

	awnserEvent := func(event *events.Event, success bool, message interface{}) {
		if event.ReturnAddr != "" {
			awnser := events.NewEvent(event.ReturnAddr, &AwnserData{
				Success: success,
				Message: message,
			})
			log.Print(awnser)
			events.Publish(awnser)
		}
	}

	go func() {
		for {
			select {
			case event := <-addUserChan:
				{
					if event.AuthLevel > 0 {
						awnserEvent(event, false, "wrong authlevel to use authentification::adduser. need authlevel 0.")
						break
					}
					if payload, ok := event.Payload.(map[string]interface{}); ok {
						username, ok1 := payload["username"].(string)
						password, ok2 := payload["password"].(string)
						authlevel_, ok3 := payload["authlevel"].(int)
						authlevel := uint8(authlevel_)
						if ok1 && ok2 && ok3 {
							if success := userManager.AddUser(username, password, authlevel); success {
								awnserEvent(event, true, "")
							} else {
								awnserEvent(event, false, "failed adding user")
							}
						} else {
							awnserEvent(event, false, "malformed payload, need 'username', 'password' and 'authlevel' fields")
						}
					} else {
						awnserEvent(event, false, "malformed payload, need 'username', 'password' and 'authlevel' fields")
					}
				}
			case event := <-delUserChan:
				{
					log.Print("got del user event")
					if event.AuthLevel > 0 {
						awnserEvent(event, false, "wrong authlevel to use authentification::deluser. need authlevel 0.")
						continue
					}
					if payload, ok := event.Payload.(map[string]interface{}); ok {
						username, ok := payload["username"].(string)
						if ok {
							if success := userManager.DelUser(username); success {
								awnserEvent(event, true, "")
							} else {
								awnserEvent(event, false, "failed deleting user")
							}
						} else {
							awnserEvent(event, false, "malformed payload, need 'username' field")
						}
					} else {
						awnserEvent(event, false, "malformed payload, need 'username' field")
					}
				}
			case event := <-checkUserChan:
				{
					if event.AuthLevel > 0 {
						awnserEvent(event, false, "wrong authlevel to use authentification::checkuser. need authlevel 0.")
						continue
					}
					if payload, ok := event.Payload.(map[string]interface{}); ok {
						username, ok1 := payload["username"].(string)
						password, ok2 := payload["password"].(string)

						if ok1 && ok2 {
							if user := userManager.CheckUser(username, password); user != nil {
								awnserEvent(event, true, user)
							} else {
								awnserEvent(event, false, nil)
							}
						} else {
							awnserEvent(event, false, "mcalformed payload, need 'username' field")
						}
					} else {
						awnserEvent(event, false, "malformed payload, need 'username' field")
					}
				}
			}
		}
	}()
}
