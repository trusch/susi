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

package apiserver

import (
	"crypto/tls"
	"encoding/json"
	"github.com/trusch/susi/config"
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/session"
	"github.com/trusch/susi/state"
	"log"
	"net"
	"testing"
)

func init() {
	events.Go()
	state.Go()
	config.Go()
	session.Go()

	state.Set("apiserver.port", "12345")
	state.Set("apiserver.tls.port", "12346")
	state.Set("apiserver.tls.cert", "/opt/cert.pem")
	state.Set("apiserver.tls.key", "/opt/key.pem")

	Go()
}

type apiserverSample struct {
	Name   string
	Input  *ApiMessage
	Output *ApiMessage
}

var samples = []apiserverSample{
	apiserverSample{
		"subscribe",
		&ApiMessage{
			Id:         12345,
			AuthLevel:  3,
			Type:       "subscribe",
			Key:        "foobar",
			ReturnAddr: "",
			Payload:    nil,
		},
		&ApiMessage{
			Id:         12345,
			AuthLevel:  0,
			Type:       "status",
			Key:        "ok",
			ReturnAddr: "",
			Payload:    "successfully subscribed to foobar",
		},
	},
	apiserverSample{
		"subscribe_after_subscribe",
		&ApiMessage{
			Id:         12345,
			AuthLevel:  3,
			Type:       "subscribe",
			Key:        "foobar",
			ReturnAddr: "",
			Payload:    nil,
		},
		&ApiMessage{
			Id:         12345,
			AuthLevel:  0,
			Type:       "status",
			Key:        "error",
			ReturnAddr: "",
			Payload:    "you are allready subscribed to foobar",
		},
	},
	apiserverSample{
		"005publish",
		&ApiMessage{
			Id:         12345,
			AuthLevel:  3,
			Type:       "publish",
			Key:        "foo",
			ReturnAddr: "",
			Payload:    nil,
		},
		&ApiMessage{
			Id:         12345,
			AuthLevel:  0,
			Type:       "status",
			Key:        "error",
			ReturnAddr: "",
			Payload:    "nobody is subscribed to foo",
		},
	},
	apiserverSample{
		"unsubscribe",
		&ApiMessage{
			Id:         12345,
			AuthLevel:  3,
			Type:       "unsubscribe",
			Key:        "foobar",
			ReturnAddr: "",
			Payload:    nil,
		},
		&ApiMessage{
			Id:         12345,
			AuthLevel:  0,
			Type:       "status",
			Key:        "ok",
			ReturnAddr: "",
			Payload:    "successfully unsubscribed from foobar",
		},
	},
	apiserverSample{
		"004unsubscribe_after_unsubscribe",
		&ApiMessage{
			Id:         12345,
			AuthLevel:  3,
			Type:       "unsubscribe",
			Key:        "foobar",
			ReturnAddr: "",
			Payload:    nil,
		},
		&ApiMessage{
			Id:         12345,
			AuthLevel:  0,
			Type:       "status",
			Key:        "error",
			ReturnAddr: "",
			Payload:    "you are not subscribed to foobar",
		},
	},
	apiserverSample{
		"006set",
		&ApiMessage{
			Id:         12345,
			AuthLevel:  3,
			Type:       "set",
			Key:        "foo",
			ReturnAddr: "",
			Payload:    "bar",
		},
		&ApiMessage{
			Id:         12345,
			AuthLevel:  0,
			Type:       "status",
			Key:        "ok",
			ReturnAddr: "",
			Payload:    "successfully saved data to foo",
		},
	},
	apiserverSample{
		"007push",
		&ApiMessage{
			Id:         12345,
			AuthLevel:  3,
			Type:       "push",
			Key:        "foo",
			ReturnAddr: "",
			Payload:    "baz",
		},
		&ApiMessage{
			Id:         12345,
			AuthLevel:  0,
			Type:       "status",
			Key:        "ok",
			ReturnAddr: "",
			Payload:    "successfully pushed data to foo",
		},
	},
	apiserverSample{
		"008enqueue",
		&ApiMessage{
			Id:         12345,
			AuthLevel:  3,
			Type:       "enqueue",
			Key:        "foo",
			ReturnAddr: "",
			Payload:    "bum",
		},
		&ApiMessage{
			Id:         12345,
			AuthLevel:  0,
			Type:       "status",
			Key:        "ok",
			ReturnAddr: "",
			Payload:    "successfully queued data to foo",
		},
	},
	apiserverSample{
		"009get",
		&ApiMessage{
			Id:         12345,
			AuthLevel:  3,
			Type:       "get",
			Key:        "foo",
			ReturnAddr: "",
			Payload:    nil,
		},
		&ApiMessage{
			Id:         12345,
			AuthLevel:  0,
			Type:       "response",
			Key:        "foo",
			ReturnAddr: "",
			Payload:    []string{"bar", "baz", "bum"},
		},
	},
	apiserverSample{
		"010pop",
		&ApiMessage{
			Id:         12345,
			AuthLevel:  3,
			Type:       "pop",
			Key:        "foo",
			ReturnAddr: "",
			Payload:    nil,
		},
		&ApiMessage{
			Id:         12345,
			AuthLevel:  0,
			Type:       "response",
			Key:        "foo",
			ReturnAddr: "",
			Payload:    "bum",
		},
	},
	apiserverSample{
		"011dequeue",
		&ApiMessage{
			Id:         12345,
			AuthLevel:  3,
			Type:       "dequeue",
			Key:        "foo",
			ReturnAddr: "",
			Payload:    nil,
		},
		&ApiMessage{
			Id:         12345,
			AuthLevel:  0,
			Type:       "response",
			Key:        "foo",
			ReturnAddr: "",
			Payload:    "bar",
		},
	},
	apiserverSample{
		"012unset",
		&ApiMessage{
			Id:         12345,
			AuthLevel:  3,
			Type:       "unset",
			Key:        "foo",
			ReturnAddr: "",
			Payload:    nil,
		},
		&ApiMessage{
			Id:         12345,
			AuthLevel:  0,
			Type:       "status",
			Key:        "ok",
			ReturnAddr: "",
			Payload:    "successfully unset data from foo",
		},
	},
	apiserverSample{
		"013undefined",
		&ApiMessage{
			Id:         12345,
			AuthLevel:  3,
			Type:       "undefined",
			Key:        "foo",
			ReturnAddr: "",
			Payload:    nil,
		},
		&ApiMessage{
			Id:         12345,
			AuthLevel:  0,
			Type:       "status",
			Key:        "error",
			ReturnAddr: "",
			Payload:    "no such request type: undefined",
		},
	},
}

func CompareApiMessages(m1, m2 *ApiMessage, t *testing.T, testcase string) {
	if m1.Id != m2.Id {
		t.Errorf("api message mismatch: testcase %v, different id's: wanted %v got %v (%v)", testcase, m1.Id, m2.Id, m2)
	}
	if m1.AuthLevel != m2.AuthLevel {
		t.Errorf("api message mismatch: testcase %v, different AuthLevel's: wanted %v got %v (%v)", testcase, m1.AuthLevel, m2.AuthLevel, m2)
	}
	if m1.Type != m2.Type {
		t.Errorf("api message mismatch: testcase %v, different Type's: wanted %v got %v (%v)", testcase, m1.Type, m2.Type, m2)
	}
	if m1.Key != m2.Key {
		t.Errorf("api message mismatch: testcase %v, different Key's: wanted %v got %v (%v)", testcase, m1.Key, m2.Key, m2)
	}
	if m1.ReturnAddr != m2.ReturnAddr {
		t.Errorf("api message mismatch: testcase %v, different ReturnAddr's: wanted %v got %v (%v)", testcase, m1.ReturnAddr, m2.ReturnAddr, m2)
	}
}

func testApiServerBasic(t *testing.T) {

	conn, err := net.Dial("tcp", "localhost:12345")
	if err != nil {
		t.Error("cant connect standard tcp socket (", err, ")")
	}

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	var response *ApiMessage

	for _, sample := range samples {
		log.Print("start sample ", sample)
		err := encoder.Encode(sample.Input)
		if err != nil {
			t.Errorf("cant send sample %v to server: %v", sample.Name, err)
		}
		err = decoder.Decode(&response)
		if err != nil {
			t.Errorf("cant decode awnser of sample %v: %v", sample.Name, err)
		}
		CompareApiMessages(sample.Output, response, t, sample.Name)
		log.Print("finished ", sample)
	}

	err = conn.Close()
	if err != nil {
		t.Errorf("Cant close standard client connection (%v)", err)
	}
}

func testTLS(t *testing.T) {

	cert, err := tls.LoadX509KeyPair("/opt/cert.pem", "/opt/key.pem")
	if err != nil {
		t.Error(err)
	}
	config := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", "localhost:12346", config)
	if err != nil {
		t.Error("cant connect to encrypted tcp socket (", err, ")")
	}

	err = conn.Close()
	if err != nil {
		t.Error("cant close encrypted tcp socket (", err, ")")
	}
}

func testPubSub(t *testing.T) {

	conn, err := net.Dial("tcp", "localhost:12345")
	if err != nil {
		t.Error("cant connect standard tcp socket (", err, ")")
	}

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	msg := &ApiMessage{
		Type: "subscribe",
		Key:  "sampleTopic",
	}

	err = encoder.Encode(msg)
	if err != nil {
		t.Errorf("Cant encode subscribe message (%v)", err)
	}
	err = decoder.Decode(msg)
	if err != nil {
		t.Errorf("Cant decode subscribe response message (%v)", err)
	}
	if msg.Type == "status" && msg.Key != "ok" {
		t.Errorf("Got error status on subscribe (%v)", msg)
	}
	msg = &ApiMessage{
		Type:    "publish",
		Key:     "sampleTopic",
		Payload: "foobar",
	}
	err = encoder.Encode(msg)
	if err != nil {
		t.Errorf("Cant encode publish message (%v)", err)
	}

	handlePacket := func(msg *ApiMessage, err error) {
		if err != nil {
			t.Errorf("Cant decode message (%v)", err)
		} else if msg.Type == "status" && msg.Key != "ok" {
			t.Errorf("Got error status (%v)", msg)
		} else if msg.Type == "event" {
			if payload, ok := msg.Payload.(string); ok {
				if payload != "foobar" {
					t.Errorf("Unexpected payload: %v", msg.Payload)
				}
			} else {
				t.Errorf("Unexpected payload type: %v (%T)", msg.Payload, msg.Payload)
			}
		}

	}
	err = decoder.Decode(msg)
	handlePacket(msg, err)
	err = decoder.Decode(msg)
	handlePacket(msg, err)

	conn.Close()
}

func TestAll(t *testing.T) {
	testApiServerBasic(t)
	testTLS(t)
	testPubSub(t)
}
