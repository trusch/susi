package main

import (
	// "../apiserver/events"
	// "../apiserver/state"
	"net/http"
	// "net"
	// "code.google.com/p/go.net/websocket"
	"log"
	"encoding/json"
	"io"
	"bytes"
	"strconv"
)

type BatchRequest map[string]map[string][]struct{
	Action string `json:"a"`
	Payload interface{} `json:"p"`
	Id int `json:"id,omitempty"`
}

type BatchResponse map[string]map[string][]struct{
	Action string `json:"a"`
	Payload interface{} `json:"p"`
	Id int `json:"id,omitempty"`
}

func PostHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		log.Print("No request could be read")
		http.Error(w, "No request could be read", http.StatusMethodNotAllowed)
		return
	}
	decoder := json.NewDecoder(io.LimitReader(req.Body,1024))
	packet := make(BatchRequest)
	err := decoder.Decode(&packet)
	if err != nil {
		log.Print("Decoding failed")
		http.Error(w, "Decoding failed", http.StatusBadRequest)
		return
	}
	for device,controllers := range packet {
		for controller,calls := range controllers {
			for idx,call := range calls {
				log.Print(device," ",controller," ",idx," ",call)
			}
		}
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

func main(){
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	http.HandleFunc("/batch",PostHandler)
	http.ListenAndServe(":8080",nil)
}

