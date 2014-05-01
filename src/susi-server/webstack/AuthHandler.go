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
	"net/http"
	"log"
	"strings"
)

type AuthHandler struct {
	defaultHandler http.Handler
}

func (ptr *AuthHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request){
	path := req.URL.Path
	if strings.HasPrefix(path,"/login") {
		log.Print(path)
	}

	ptr.defaultHandler.ServeHTTP(resp,req)
}