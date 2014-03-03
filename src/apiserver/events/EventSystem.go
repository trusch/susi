package events

/*
This provides a Publish-Subscribe server for the system
*/

import (
	"log"
	"time"
	"errors"
)

type CommandType uint8

const (
	SUBSCRIBE CommandType = iota
	UNSUBSCRIBE
	PUBLISH
)

type command struct {
	Type    CommandType
	Topic   string
	Payload interface{}
	Result  chan interface{}
}

type unsubscribeResult struct {
	EventChan chan interface{}
	CloseChan chan bool
}

type EventSystem struct {
	cmdChan chan *command
	topics  map[string]map[int64]chan interface{}
}

var eventSystem *EventSystem

func init() {
	eventSystem = new(EventSystem)
	eventSystem.cmdChan = make(chan *command, 10)
	eventSystem.topics = make(map[string]map[int64]chan interface{})
	go func() {
		for cmd := range eventSystem.cmdChan {
			switch cmd.Type {
			case SUBSCRIBE:
				{
					eventChan := make(chan interface{},10)
					unsubscribeChan := make(chan bool)
					channelMap := eventSystem.topics[cmd.Topic]
					if channelMap==nil {
						tmp := make(map[int64]chan interface{})
						eventSystem.topics[cmd.Topic] = tmp
						channelMap = tmp
					}
					now := time.Now().UnixNano();
					channelMap[now] = eventChan
					log.Print("subscribtion to ",cmd.Topic)
					go func(){
						<-unsubscribeChan
						log.Print("delete subscribtion to ",cmd.Topic)
						delete(channelMap,now)
					}()			
					cmd.Result <- unsubscribeResult{
						EventChan: eventChan,
						CloseChan: unsubscribeChan,
					}
				}
			case PUBLISH:
				{
					chans := eventSystem.topics[cmd.Topic]
					for key, outChan := range chans {
						err := safeSend(outChan,cmd.Payload)
						if err!=nil {
							log.Print(err)
							delete(chans,key)
						}
					}
				}
			}
		}
	}()
	log.Print("successfully started EventSystem")
}

/*
returns error when chanel is closed
*/
func safeSend(c chan interface{}, t interface{}) (err error) {
	defer func() {
		if x := recover(); x != nil {
			err = errors.New("SendError")
		}
	}()
	c <- t
	return
}

/*
Global publish function
*/
func Publish(topic string, payload interface{}) {
	command := &command{
		Type:    PUBLISH,
		Topic:   topic,
		Payload: payload,
	}
	eventSystem.cmdChan <- command
}

/*
Global subscribe function
*/
func Subscribe(topic string) (eventChannel chan interface{},closeChannel chan bool) {
	command := &command{
		Type:    SUBSCRIBE,
		Topic:   topic,
		Result:  make(chan interface{}),
	}
	eventSystem.cmdChan <- command
	res_ := <-command.Result
	res := res_.(unsubscribeResult)
	eventChannel = res.EventChan
	closeChannel = res.CloseChan
	return
}

