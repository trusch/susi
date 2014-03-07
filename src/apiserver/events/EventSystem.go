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

func (ptr *EventSystem) subscribe(topic string) (eventChannel chan interface{},closeChannel chan bool) {
	eventChannel = make(chan interface{},10)
	closeChannel = make(chan bool)
	now := time.Now().UnixNano();
	channelMap := ptr.topics[topic]
	if channelMap==nil {
		tmp := make(map[int64]chan interface{})
		ptr.topics[topic] = tmp
		channelMap = tmp
	}
	channelMap[now] = eventChannel
	log.Print("subscribed to ",topic," (",now,")")
	go func(){
		<-closeChannel
		log.Print("unsubscribed from ",topic," (",now,")")
		delete(channelMap,now)
	}()
	return eventChannel,closeChannel
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
					ec,cc := eventSystem.subscribe(cmd.Topic)
					cmd.Result <- unsubscribeResult{
						EventChan: ec,
						CloseChan: cc,
					}
				}
			case PUBLISH:
				{
					chans := eventSystem.topics[cmd.Topic]
					if len(chans) == 0 {
						cmd.Result <- false
						break
					}
					for key, outChan := range chans {
						err := safeSend(outChan,cmd.Payload)
						if err!=nil {
							log.Print(err)
							delete(chans,key)
						}
					}
					cmd.Result <- true
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
func Publish(topic string, payload interface{}) bool{
	command := &command{
		Type:    PUBLISH,
		Topic:   topic,
		Payload: payload,
		Result:  make(chan interface{}),
	}
	eventSystem.cmdChan <- command
	res := (<-command.Result).(bool)
	return res
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

