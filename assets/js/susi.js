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

var susi = {
	internal : {
		sendPostMessage: function(url,data,callback,errorcallback){
			callback = callback || susi.internal.log
			errorcallback = errorcallback || susi.internal.log
			var xmlhttp;
			if (window.XMLHttpRequest) {// code for IE7+, Firefox, Chrome, Opera, Safari
				xmlhttp = new XMLHttpRequest();
			}else{// code for IE6, IE5
				xmlhttp = new ActiveXObject("Microsoft.XMLHTTP");
			}
			xmlhttp.onreadystatechange=function(){
				if (xmlhttp.readyState==4){
					if (xmlhttp.status == 200) {
						if (callback !== undefined) {
							callback(xmlhttp.response,200)
						}
					}else{
						if (errorcallback !== undefined) {
							errorcallback(xmlhttp.response,xmlhttp.status)
						}
					}
				}
			};
			xmlhttp.open("POST",url,true);
			xmlhttp.send(JSON.stringify(data));
		},

		log: function(data){
			console.log(data)
		}
	},

	auth: {
		login: function(username,password){
			var onSuccess = function(){
				console.log("successfully logged in as "+username)
			}
			var onError = function(){
				console.log("failed logging in as "+username)
			} 
			susi.internal.sendPostMessage("/auth/login",{username: username,password: password},
				onSuccess,
				onError);
		},

		logout: function() {
			susi.internal.sendPostMessage("/auth/logout");	
		},

		keepAlive: function(){
			susi.internal.sendPostMessage("/auth/keepalive");	
		},

		info: function(){
			susi.internal.sendPostMessage("/auth/info",null,susi.internal.log);	
		},

	},

	events: {

		subscriptions: {},

		publish: function(key,data,authlevel,returnaddr){
			authlevel = authlevel || 0
			var msg = {
				key: key,
				payload: data,
				authlevel: authlevel,
				returnaddr: returnaddr
			}
			susi.internal.sendPostMessage("/events/publish",msg)
		},
		subscribe: function(key,callback,authlevel){
			authlevel = authlevel || 0
			var msg = {
				key: key,
				authlevel: authlevel
			}
			susi.events.subscriptions[key] = susi.events.subscriptions[key] || []
			susi.events.subscriptions[key].push(callback)
			susi.internal.sendPostMessage("/events/subscribe",msg)
		},
		get: function(){
			susi.internal.sendPostMessage("/events/get",null,function(result){
				result = JSON.parse(result)
				if (result !== null) {
					for (var i = result.length - 1; i >= 0; i--) {
						var evt = result[i]
						var callbacks = susi.events.subscriptions[evt.topic]
						if (callbacks != null ){
							for (var j = callbacks.length - 1; j >= 0; j--) {
								callbacks[j](evt)
							};
						}
					};
				}
			})
		}
	},

}