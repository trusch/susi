package autodiscovery

import (
	"net"
	"log"
	"flag"
	"strings"
)

var ownIPAddr = flag.String("addr","","your addr in form 192.0.0.123:55555")

func GetOwnAddr(ownPort string) string {
	if *ownIPAddr=="" {
		addrs,err := net.InterfaceAddrs();
		if err!=nil {
			log.Fatal(err)
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
			*ownIPAddr = parts[0]+":"+ownPort
			break
		}
	}
	return *ownIPAddr
} 

func ListenForBroadcastMessage(port int){
	socket, err := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(255,255,255,255),
		Port: port,
	})

	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}

	go func(){
		defer socket.Close()
		buff := make([]byte, 4096)
		for {
			read, err := socket.Read(buff[0:])
			if err != nil {
				log.Fatal(err)
			}
			msgType := buff[0]
			msg := string(buff[1:read])
			switch msgType {
				case 'r': {
					log.Println(msg)

				}
			}
		}
	}()	
}

func StartAutodiscovery(){
	ListenForBroadcastMessage(4242);
	conn,err := net.Dial("udp","255.255.255.255:4242")
	if err!=nil {
		log.Fatal(err)
	}
	conn.Write([]byte("rFoobar!"))
	log.Print(GetOwnAddr("12345"))
	conn.Close()
}
