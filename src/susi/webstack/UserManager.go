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
	"crypto/sha512"
	"encoding/base64"
	"bytes"
	"flag"
	"os"
	"log"
	"encoding/json"
)

var hashRounds = flag.Int("webstack.hashRounds",64,"How many hash rounds to perform")
var usersFile = flag.String("webstack.usersFile","users.json","The file where the login data will be saved")

func NewUserManager() *UserManager {
	ptr := new(UserManager)
	ptr.cmds = make(chan userManagerCommand,10)
	ptr.users = make([]User,0,32)
	ptr.Load()
	go ptr.backend()
	return ptr
}
func (ptr *UserManager) AddUser(name,password string) bool {
	cmd := userManagerCommand{
		Type: ADDUSER,
		Return: make(chan bool),
		User: User{
			Username: name,
			Password: password,
		},
	}
	ptr.cmds <- cmd
	return <-cmd.Return
}

func (ptr *UserManager) DelUser(name string) bool {
	cmd := userManagerCommand{
		Type: DELUSER,
		Return: make(chan bool),
		User: User{
			Username: name,
		},
	}
	ptr.cmds <- cmd
	return <- cmd.Return
}

func (ptr *UserManager) CheckUser(name,password string) bool {
	cmd := userManagerCommand{
		Type: CHECKUSER,
		Return: make(chan bool),
		User: User{
			Username: name,
			Password: password,
		},
	}
	ptr.cmds <- cmd
	return <-cmd.Return
}

type User struct {
	Username string
	Password string
}

func (user *User) HashPassword() {
	for i:=0;i<*hashRounds;i++ {
		buff := &bytes.Buffer{}
		hash := sha512.New()
		encoder := base64.NewEncoder(base64.StdEncoding,buff)
		hash.Write([]byte(user.Password))
		encoder.Write(hash.Sum(make([]byte,0,20)))
		user.Password = buff.String()
	}
}

type userManagerCommandType int
const (
	ADDUSER userManagerCommandType = iota
	DELUSER
	CHECKUSER
)

type userManagerCommand struct {
	User User
	Type userManagerCommandType
	Return chan bool
}


type UserManager struct {
	cmds chan userManagerCommand
	users []User
}

func (manager *UserManager) backend() {
	MAINLOOP:
	for cmd := range manager.cmds {
		switch cmd.Type {
			case ADDUSER: {
				for _,user := range manager.users {
					if user.Username == cmd.User.Username {
						cmd.Return <- false
						continue MAINLOOP
					}
				}
				cmd.User.HashPassword()
				manager.users = append(manager.users,cmd.User)
				manager.Save()
				cmd.Return <- true
			}
			case DELUSER:{	
				for idx,user := range manager.users {
					if user.Username == cmd.User.Username {
						manager.users = append(manager.users[:idx],manager.users[idx+1:]...)
						manager.Save()
						cmd.Return <- true
						continue MAINLOOP
					}
				}
				cmd.Return <- false
			}
			case CHECKUSER: {
				cmd.User.HashPassword()
				for _,user := range manager.users {
					if user.Username == cmd.User.Username {
						if user.Password == cmd.User.Password {
							cmd.Return <- true
							continue MAINLOOP
						}else{
							cmd.Return <- false
							continue MAINLOOP
						}
					}
				}
				cmd.Return <- false
			}
		}
	}
}

func (ptr *UserManager) Load(){
	f,err := os.Open(*usersFile)
	if err!=nil {
		log.Print(err)
		return
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&ptr.users)
	if err!=nil {
		log.Print(err)
		return
	}
}

func (ptr *UserManager) Save(){
	f,err := os.Create(*usersFile)
	if err!=nil {
		log.Print(err)
		return
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	err = encoder.Encode(ptr.users)
	if err!=nil {
		log.Print(err)
		return
	}
}
