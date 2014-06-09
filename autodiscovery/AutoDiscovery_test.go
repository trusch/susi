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
	"github.com/trusch/susi/events"
	"github.com/trusch/susi/state"
	"testing"
)

func TestAutodiscoveryManager(t *testing.T) {

	events.Go()
	state.Go()

	ch, _ := events.Subscribe("hosts::*", 0)

	NewAutodiscoveryManager("224.0.0.23:42424", GetOwnAddr("12345"))
	NewAutodiscoveryManager("224.0.0.23:42424", GetOwnAddr("12346"))

	event := <-ch
	if event.Topic != "hosts::new" {
		t.Error("got wrong event %v expected hosts::new", event.Topic)
	}
	event = <-ch
	if event.Topic != "hosts::new" {
		t.Error("got wrong event %v expected hosts::new", event.Topic)
	}

	event = events.NewEvent("hosts::lost", "127.0.0.1:232323")
	event.AuthLevel = 0
	events.Publish(event)

	state.Set("autodiscovery.mcastAddr", "224.0.0.23:42424")
	state.Set("apiserver.port", "4242")
	state.Set("autodiscovery.names", []string{"foo", "bar"})

	Go()

}
