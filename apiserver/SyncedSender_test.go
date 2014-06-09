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
	"net"
	"testing"
)

func TestSyncedSender(t *testing.T) {

	type testSample struct {
		Input  interface{}
		Output []byte
	}

	type sampleType struct {
		A int    `json:"a"`
		B string `json:"b"`
	}

	samples := map[string]testSample{
		"byteArray": testSample{
			[]byte{65, 66, 67},
			[]byte{65, 66, 67},
		},
		"string": testSample{
			"hello world",
			[]byte("hello world"),
		},
		"sampleType": testSample{
			sampleType{23, "foobar"},
			[]byte(`{"a":23,"b":"foobar"}` + "\n"),
		},
	}

	conn, end := net.Pipe()
	sender := NewSyncedSender(conn)
	buff := make([]byte, 1024)

	for testName, sample := range samples {
		err := sender.Send(sample.Input)
		if err != nil {
			t.Error("got error while sending byte array")
		}
		bs, err := end.Read(buff[0:])
		if err != nil || bs != len(sample.Output) {
			t.Errorf("Failed while testing %v. Expected %v byte got %v byte.", testName, len(sample.Output), len(buff[:bs]))

		}
		for idx, val := range sample.Output {
			if val != buff[idx] {
				t.Errorf("Failed while testing %v. Expected %v got %v. Diff in byte %v.", testName, string(sample.Output), string(buff[:bs]), idx)
				break
			}
		}
	}

	sender.Close()
	err := sender.Send("foo")
	if err == nil {
		t.Error("No error after sending on closed sender")
	}
}
