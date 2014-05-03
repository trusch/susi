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

package autodiscovery

import (
	"../events"
	"../state"
	"flag"
	"log"
	"net"
	"strings"
	"time"
)

var autodiscoveryMulticastAddr = flag.String("autodiscovery.mcastAddr", "224.0.0.23:42424", "the autodiscovery multicast addr")
var autodiscoveryTcpPort = flag.String("autodiscovery.port", "42424", "the autodiscovery tcp port")

type AutodiscoveryManager struct {
	InputNew  chan string
	InputLost chan string
	Hosts     map[string]bool
}

func (ptr *AutodiscoveryManager) backend() {
	ptr.InputNew = make(chan string, 10)
	ptr.InputLost = make(chan string, 10)
	ptr.Hosts = make(map[string]bool)
	ptr.ListenForMulticastMessage()
	ptr.ListenForDirectMessage()
	own := ptr.GetOwnAddr(state.Get("apiserver.port").(string))
	ptr.SendMulticastMessage(ptr.GetOwnAddr(state.Get("autodiscovery.port").(string)))

	for {
		select {
		case addr := <-ptr.InputNew:
			{
				if _, ok := ptr.Hosts[addr]; !ok {
					if addr == own {
						continue
					}
					ptr.Hosts[addr] = true
					event := events.NewEvent("hosts::new", addr)
					event.AuthLevel = 0
					events.Publish(event)
				}
			}
		case addr := <-ptr.InputLost:
			{
				if addr == own {
					continue
				}
				delete(ptr.Hosts, addr)
				event := events.NewEvent("hosts::lost", addr)
				event.AuthLevel = 0
				events.Publish(event)
			}
		}
	}
}

func (ptr *AutodiscoveryManager) GetOwnAddr(ownPort string) string {
	ownIPAddr := ""
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		//log.Println(err)
		return ""
	}
	blacklist := []string{
		"127.",
		"::1",
		"fe80:",
	}

OUTERLOOP:
	for _, addr := range addrs {
		for _, black := range blacklist {
			if strings.HasPrefix(addr.String(), black) {
				continue OUTERLOOP
			}
		}
		parts := strings.Split(addr.String(), "/")
		ownIPAddr = parts[0] + ":" + ownPort
		break
	}

	return ownIPAddr
}

func (ptr *AutodiscoveryManager) ListenForMulticastMessage() {
	ownApiAddr := ptr.GetOwnAddr(state.Get("apiserver.port").(string))
	ownDiscoveryAddr := ptr.GetOwnAddr(state.Get("autodiscovery.port").(string))

	mcaddr, err := net.ResolveUDPAddr("udp", state.Get("autodiscovery.mcastAddr").(string))
	if err != nil {
		//log.Println(err)
		return
	}
	socket, err := net.ListenMulticastUDP("udp4", nil, mcaddr)
	if err != nil {
		//log.Println(err)
		return
	}
	go func() {
		defer socket.Close()
		buff := make([]byte, 4096)
		for {
			read, err := socket.Read(buff[0:])
			if err != nil {
				//log.Println(err)
				return
			}
			addr := string(buff[:read])
			if addr != ownDiscoveryAddr {
				ptr.SendDirectMessage(addr, ownApiAddr)
			}
		}
	}()
}

func (ptr *AutodiscoveryManager) ListenForDirectMessage() {
	ownAddr := ptr.GetOwnAddr(state.Get("apiserver.port").(string))
	accp, err := net.Listen("tcp", ":"+state.Get("autodiscovery.port").(string))
	if err != nil {
		log.Println(err)
		return
	}
	go func() {
		for {
			conn, err := accp.Accept()
			if err != nil {
				return
			} else {
				//Got a direct message
				go func() {
					defer conn.Close()
					buff := make([]byte, 4096)
					bs, err := conn.Read(buff)
					if err != nil {
						return
					}
					hostAddr := string(buff[:bs])
					conn.Write([]byte(ownAddr))
					ptr.InputNew <- hostAddr
					for {
						bs, err := conn.Read(buff)
						if err != nil {
							event := events.NewEvent("hosts::lost", hostAddr)
							event.AuthLevel = 0
							events.Publish(event)
							return
						}
						_, err = conn.Write(buff[:bs])
						if err != nil {
							event := events.NewEvent("hosts::lost", hostAddr)
							event.AuthLevel = 0
							events.Publish(event)
							return
						}
					}
				}()
			}
		}
	}()
}

func (ptr *AutodiscoveryManager) SendMulticastMessage(msg string) {
	addr := state.Get("autodiscovery.mcastAddr").(string)
	conn, err := net.Dial("udp", addr)
	if err != nil {
		log.Println(err)
		return
	}
	conn.Write([]byte(msg))
	conn.Close()
}

func (ptr *AutodiscoveryManager) SendDirectMessage(addr, msg string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		//log.Println(err)
		return
	}
	_, err = conn.Write([]byte(msg))
	if err != nil {
		//log.Println(err)
		conn.Close()
		return
	}
	buff := make([]byte, 1024)
	bs, err := conn.Read(buff)
	if err != nil {
		//log.Println(err)
		conn.Close()
		return
	}
	addr = string(buff[:bs])
	ptr.InputNew <- addr
	go func() {
		defer conn.Close()
		for {
			time.Sleep(1 * time.Second)
			_, err = conn.Write([]byte("ping"))
			if err != nil {
				//log.Println(err)
				ptr.InputLost <- addr
				return
			}
			_, err = conn.Read(buff[0:])
			if err != nil {
				//log.Println(err)
				ptr.InputLost <- addr
				return
			}
		}
	}()
}

func NewAutodiscoveryManager() *AutodiscoveryManager {
	res := new(AutodiscoveryManager)
	go res.backend()
	return res
}

func Go() {
	flag.Parse()
	NewAutodiscoveryManager()
}
