package dns

import (
	"log"
	"net"

	"github.com/TasSM/labns/internal/defs"
	"golang.org/x/net/dns/dnsmessage"
)

type DNSServiceConstruct struct {
	conn  *net.UDPConn
	state defs.StateMap
}

func CreateNewDNSService(c *net.UDPConn) defs.DNSService {
	return &DNSServiceConstruct{
		conn:  c,
		state: CreateNewStateMap(),
	}
}

func (svc DNSServiceConstruct) Start() {
	log.Printf("Starting Listener service on port %s", svc.conn.LocalAddr().String())
	for {
		buf := make([]byte, 512)
		_, addr, _ := svc.conn.ReadFromUDP(buf)
		var m dnsmessage.Message
		err := m.Unpack(buf)
		if err != nil {
			log.Printf("Invalid DNS message received from addr %s: $v", addr, buf)
			continue
		}
		packed, _ := m.Pack()
		if m.Header.Response {
			log.Printf("received response")
			key, err := HashMessageFields(&m)
			if err != nil {
				log.Fatalf("error encountered: %v", err)
			}
			go func() {
				adlist, err := svc.state.RetrieveAndDelete(key)
				if err != nil {
					log.Fatalf("Received invalid key from upstream request")
				}
				for _, v := range adlist {
					go svc.conn.WriteToUDP(packed, v)
				}
			}()
			continue
		} else {
			//check local records
			//check cache
			//forward
			log.Printf("received request")
			key, err := HashMessageFields(&m)
			if err != nil {
				log.Fatalf("error encountered: %v", err)
			}
			svc.state.AddRequestor(key, addr)
			resolver := net.UDPAddr{IP: net.IP{1, 1, 1, 1}, Port: 53}
			go svc.conn.WriteToUDP(packed, &resolver)
		}
	}
}
