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
	"code.google.com/p/go.net/websocket"
	"crypto/tls"
	"flag"
	"github.com/trusch/susi/apiserver"
	"github.com/trusch/susi/state"
	"log"
	"net/http"
	"strconv"
	"time"
)

var httpAddr = flag.String("webstack.addr", ":8080", "The web addr")
var tlsCert = flag.String("webstack.tls.cert", "", "The TLS certificate")
var tlsKey = flag.String("webstack.tls.key", "", "The TLS key")
var assetRoot = flag.String("webstack.assets", "./assets", "The root directory for assets")

func Go() {
	addr := state.Get("webstack.addr").(string)
	certFile := state.Get("webstack.tls.cert").(string)
	keyFile := state.Get("webstack.tls.key").(string)
	assetsDir := state.Get("webstack.assets").(string)

	if addr == "" {
		log.Print("not starting webstack. no address given.")
		return
	}

	eventsHandler := NewEventsHandler()

	handler := http.NewServeMux()
	handler.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(assetsDir))))
	handler.Handle("/events/", eventsHandler)
	handler.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		req := ws.Request()
		sessionId_, _ := strconv.Atoi(req.Header.Get("sessionid"))
		sessionId := uint64(sessionId_)
		apiserver.HandleConnection(ws, sessionId)
	}))
	handler.Handle("/", http.RedirectHandler("/assets/main.html", http.StatusMovedPermanently))

	authHandler := NewAuthHandler(handler)

	server := &http.Server{
		Addr:           addr,
		Handler:        authHandler,
		ReadTimeout:    2 * time.Second,
		WriteTimeout:   2 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if certFile != "" && keyFile != "" {
		_, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			log.Print("No valid tls cert/key", err)
			certFile = ""
			keyFile = ""
		}
	}

	if certFile != "" && keyFile != "" {
		go func() {
			log.Fatal(server.ListenAndServeTLS(certFile, keyFile))
		}()
		log.Print("Successfully started HTTP Server with TLS encryption on ", server.Addr)
	} else {
		go func() {
			log.Fatal(server.ListenAndServe())
		}()
		log.Print("Successfully started HTTP Server on ", server.Addr)
	}
}
