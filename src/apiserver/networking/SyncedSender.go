package networking

import (
	"encoding/json"
	"errors"
	"log"
	"net"
)

type closeType struct{}

type SyncedSender struct {
	Conn   net.Conn
	in     chan interface{}
	closed bool
}

func NewSyncedSender(conn net.Conn) *SyncedSender {
	sw := &SyncedSender{
		Conn: conn,
		in:   make(chan interface{}, 10),
	}
	go func() {
		encoder := json.NewEncoder(conn)
		for {
			data := <-sw.in
			var err error
			switch data.(type) {
			case string:
				{
					_, err = conn.Write([]byte(data.(string)))
				}
			case []byte:
				{
					_, err = conn.Write(data.([]byte))
				}
			case closeType:
				{
					return
				}
			case interface{}:
				{
					err = encoder.Encode(data)
				}
			}
			if err != nil {
				log.Print(err)
				close(sw.in)
				sw.closed = true
				return
			}
		}
	}()
	return sw
}
func (sw *SyncedSender) Send(data interface{}) error {
	sw.in <- data
	if sw.closed {
		return errors.New("error while sending")
	}
	return nil
}
func (sw *SyncedSender) Close() {
	sw.in <- closeType{}
}
