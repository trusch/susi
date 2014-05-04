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
	"../state"
	"../apiserver"
	"crypto/tls"
	"flag"
	"log"
	"net/http"
	"time"
	"strconv"
	"code.google.com/p/go.net/websocket"
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

	eventsHandler := NewEventsHandler()

	handler := http.NewServeMux()
	handler.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(assetsDir))))
	handler.Handle("/events/", eventsHandler)
	handler.Handle("/ws",websocket.Handler(func(ws *websocket.Conn){
		req := ws.Request()
		authlevel_, _ := strconv.Atoi(req.Header.Get("authlevel"))
		authlevel := uint8(authlevel_)
		apiserver.HandleConnection(ws,authlevel)
	}))
	handler.Handle("/", http.RedirectHandler("/assets/main.html", http.StatusMovedPermanently))
	
	authHandler := NewAuthHandler(handler)

	server := &http.Server{
		Addr:           addr,
		Handler:        authHandler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
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
