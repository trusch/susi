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
	"flag"
	"github.com/trusch/susi/autodiscovery/remoteeventcollector"
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/state"
	"net"
	"strings"
)

var autodiscoveryMulticastAddr = flag.String("autodiscovery.mcastAddr", "224.0.0.23:42424", "the autodiscovery multicast addr")
var autodiscoveryTcpPort = flag.String("autodiscovery.port", "42424", "the autodiscovery tcp port")

type AutodiscoveryManager struct {
	InputNew  chan string
	InputLost chan *events.Event
	Hosts     map[string]bool

	mcastAddr     string
	apiserverAddr string
}

func (ptr *AutodiscoveryManager) backend() {
	ptr.InputNew = make(chan string, 10)
	ptr.InputLost, _ = events.Subscribe("hosts::lost", 0)
	ptr.Hosts = make(map[string]bool)
	ptr.ListenForMulticastMessage()
	ptr.SendMulticastMessage(ptr.apiserverAddr)

	for {
		select {
		case addr := <-ptr.InputNew:
			{
				if _, ok := ptr.Hosts[addr]; !ok {
					if _, ok := ptr.Hosts[addr]; !ok && addr != ptr.apiserverAddr {
						ptr.Hosts[addr] = true
						ptr.SendMulticastMessage(ptr.apiserverAddr)
						event := events.NewEvent("hosts::new", addr)
						event.AuthLevel = 0
						events.Publish(event)
					}
				}
			}
		case event := <-ptr.InputLost:
			{
				delete(ptr.Hosts, event.Payload.(string))
			}
		}
	}
}

func (ptr *AutodiscoveryManager) ListenForMulticastMessage() {

	mcaddr, err := net.ResolveUDPAddr("udp", ptr.mcastAddr)
	if err != nil {
		return
	}
	socket, err := net.ListenMulticastUDP("udp4", nil, mcaddr)
	if err != nil {
		return
	}
	go func() {
		defer socket.Close()
		buff := make([]byte, 4096)
		for {
			read, err := socket.Read(buff[0:])
			if err != nil {
				return
			}
			addr := string(buff[:read])
			ptr.InputNew <- addr
		}
	}()
}

func (ptr *AutodiscoveryManager) SendMulticastMessage(msg string) {
	conn, err := net.Dial("udp", ptr.mcastAddr)
	if err != nil {
		return
	}
	conn.Write([]byte(msg))
	conn.Close()
}

func NewAutodiscoveryManager(mcastAddr, apiserverAddr string) *AutodiscoveryManager {
	res := new(AutodiscoveryManager)
	res.mcastAddr = mcastAddr
	res.apiserverAddr = apiserverAddr
	go res.backend()
	return res
}

func GetOwnAddr(ownPort string) string {
	ownIPAddr := ""
	addrs, err := net.InterfaceAddrs()
	if err != nil {
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

func Go() {
	flag.Parse()
	mcastAddr := state.Get("autodiscovery.mcastAddr").(string)
	apiserverPort := state.Get("apiserver.port").(string)
	apiserverAddr := GetOwnAddr(apiserverPort)
	names := []string{"all"}
	if namesFromConfig, ok := state.Get("autodiscovery.names").([]string); ok {
		names = append(names, namesFromConfig...)
	}
	remoteeventcollector.New(names)
	NewAutodiscoveryManager(mcastAddr, apiserverAddr)
}
