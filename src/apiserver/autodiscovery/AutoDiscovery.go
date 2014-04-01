package autodiscovery

import (
	"net"
	"log"
	"flag"
	"strings"
	"strconv"
)

var autodiscoveryMuticastPort = flag.Int("autodiscoveryMcastPort",4242,"the autodiscovery multicast port")
var autodiscoveryTcpPort = flag.Int("autodiscoveryTcpPort",4242,"the autodiscovery tcp port")
var autodiscoveryAnouncePort = flag.Int("autodiscoveryAnouncePort",12345,"the autodiscovery tcp port")

type AutodiscoveryManager struct {
	Input chan string
	Hosts map[string]bool
}

func (ptr *AutodiscoveryManager) backend(){
	ptr.Input = make(chan string,10)
	ptr.Hosts = make(map[string]bool)
	ptr.ListenForMulticastMessage(*autodiscoveryMuticastPort)
	ptr.ListenForDirectMessage(*autodiscoveryTcpPort)
	own := ptr.GetOwnAddr(strconv.Itoa(*autodiscoveryAnouncePort))
	ptr.SendMulticastMessage(*autodiscoveryMuticastPort,ptr.GetOwnAddr(strconv.Itoa(*autodiscoveryTcpPort)))
	for str := range ptr.Input {
		if _,ok := ptr.Hosts[str]; !ok {
			if str==own {
				continue
			}
			ptr.Hosts[str] = true
			log.Print("new host: ",str)
		}
	}
}

func NewAutodiscoveryManager() *AutodiscoveryManager{
	res := new(AutodiscoveryManager)
	go res.backend();
	return res
}


func (ptr *AutodiscoveryManager) GetOwnAddr(ownPort string) string {
	ownIPAddr := ""
	addrs,err := net.InterfaceAddrs();
	if err!=nil {
		log.Println(err)
		return ""
	}
	blacklist := []string{
		"127.",
		"::1",
		"fe80:",
	}

	OUTERLOOP:
	for _,addr := range addrs {
		for _,black := range blacklist {
			if strings.HasPrefix(addr.String(),black){
				continue OUTERLOOP
			}
		}
		parts := strings.Split(addr.String(),"/")
		ownIPAddr = parts[0]+":"+ownPort
		break
	}

	return ownIPAddr
} 

func (ptr *AutodiscoveryManager) ListenForMulticastMessage(port int){
	ownAddr := ptr.GetOwnAddr(strconv.Itoa(*autodiscoveryAnouncePort))
	mcaddr, err := net.ResolveUDPAddr("udp", "224.0.0.23:"+strconv.Itoa(port))
	if err != nil {
		log.Println(err)
		return
	}
	socket, err := net.ListenMulticastUDP("udp4", nil,mcaddr)
	if err != nil {
		log.Println(err)
		return
	}
	go func(){
		defer socket.Close()
		buff := make([]byte, 4096)
		for {
			read, err := socket.Read(buff[0:])
			if err != nil {
				log.Println(err)
				return
			}
			addr := string(buff[:read])
			ptr.SendDirectMessage(addr,ownAddr)
		}
	}()	
}

func (ptr *AutodiscoveryManager) ListenForDirectMessage(port int){
	ownAddr := ptr.GetOwnAddr(strconv.Itoa(*autodiscoveryAnouncePort))
	accp, err := net.Listen("tcp",":"+strconv.Itoa(port))
	if err != nil {
		log.Println(err)
		return
	}
	go func(){
		for {
			conn,err := accp.Accept()
			if err!=nil {
				log.Println(err)
				return
			}else{
				go func(){
					defer conn.Close()
					buff := make([]byte, 4096)
					bs,err := conn.Read(buff)
					if err!=nil {
						log.Println(err)
						return
					}
					msg := string(buff[:bs])
					conn.Write([]byte(ownAddr))
					ptr.Input <- msg
				}()
			}
		}
	}()	
}

func (ptr *AutodiscoveryManager) SendMulticastMessage(port int,msg string){
	portStr := strconv.Itoa(port)
	conn,err := net.Dial("udp","224.0.0.23:"+portStr)
	if err!=nil {
		log.Println(err)
		return
	}
	conn.Write([]byte(msg))
	conn.Close()
}

func (ptr *AutodiscoveryManager) SendDirectMessage(addr, msg string){
	conn,err := net.Dial("tcp",addr)
	if err!=nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	_,err = conn.Write([]byte(msg))
	if err!=nil {
		log.Println(err)
		return
	}
	buff := make([]byte,1024)
	bs,err := conn.Read(buff)
	if err!=nil {
		log.Println(err)
		return
	}
	msg = string(buff[:bs])
	ptr.Input <- msg
}

func Run(){
	flag.Parse()
	NewAutodiscoveryManager()	
}
