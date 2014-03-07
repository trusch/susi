package main

import (
	"../apiserver/events"
	// "../apiserver/state"
	"net/http"
	// "net"
	// "code.google.com/p/go.net/websocket"
	"log"
	"encoding/json"
	"io"
	"bytes"
	"strconv"
	"time"
	"errors"
)

type BatchMessage map[string]map[string][]*struct{
	Action string `json:"a,omitempty"`
	Payload interface{} `json:"p,omitempty"`
	Result interface{} `json:"result,omitempty"`
	Timeout int64 `json:"timeout,omitempty"`
	Id int `json:"id,omitempty"`
}

func PostHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		log.Print("No request could be read")
		http.Error(w, "No request could be read", http.StatusMethodNotAllowed)
		return
	}
	decoder := json.NewDecoder(io.LimitReader(req.Body,1024))
	packet := make(BatchMessage)
	err := decoder.Decode(&packet)
	if err != nil {
		log.Print("Decoding failed")
		http.Error(w, "Decoding failed", http.StatusBadRequest)
		return
	}
	ready := make(chan bool,10)
	toWait := 0
	for device,controllers := range packet {
		for controller,calls := range controllers {
			for idx,call := range calls {
				toWait += 1
				log.Print(device," ",controller," ",idx," ",call," ",toWait)
				if call.Timeout == 0 {
					call.Timeout = 200
				}
				timeout := time.Duration(call.Timeout) * time.Millisecond
				call_ := call
				go func(){
					result,err := GetControllerCallAwnser(controller,call.Action,call.Payload,timeout)
					if err!=nil {
						call_.Result = err.Error()
					}else{
						call_.Result = result
					}
					call_.Action = ""
					call_.Payload = nil
					call_.Timeout = 0
					ready <- true
				}()
			}
		}
	}
	for i:=0;i<toWait;i++ {
		<-ready
	}
	var buff bytes.Buffer
	encoder := json.NewEncoder(&buff)
	err = encoder.Encode(&packet)
	if err!=nil {
		log.Print("Encoding failed")
		http.Error(w, "Encoding failed", http.StatusInternalServerError)
	}
	w.Header().Set("content-length", strconv.FormatInt(int64(buff.Len()),10))
	w.Write(buff.Bytes())
}

func GetControllerCallAwnser(controller,call string, payload interface{}, timeout time.Duration) (result interface{},err error) {
	log.Print("getControllerCallAwnser: ",controller," ",call," ",payload," ",timeout)
	requestTopic := "controller::"+controller+"::"+call
	resultTopic := strconv.FormatInt(time.Now().UnixNano(),10)
	resultChan,unsubChan := events.Subscribe(resultTopic)
	defer func(){unsubChan <- true}()
	packet := make(map[string]interface{})
	packet["payload"] = payload
	packet["resultTopic"] = resultTopic
	if ok := events.Publish(requestTopic,packet); !ok {
		log.Print("no such topic")
		err = errors.New("no such topic: "+requestTopic)
		return
	}
	log.Print(timeout)
	select {
		case result = <-resultChan:{
		}
		case <-time.After(timeout):{
			log.Print("timeout!")
			err = errors.New("timeout while calling "+requestTopic)
		}
	}
	return
}

func main(){
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	http.HandleFunc("/batch",PostHandler)
	http.ListenAndServe(":8080",nil)
}

