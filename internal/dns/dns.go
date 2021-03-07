package dns

import (
	"log"
	"net"

	"github.com/TasSM/labns/internal/defs"
	"golang.org/x/net/dns/dnsmessage"
)

type DNSServiceConstruct struct {
	conn         *net.UDPConn
	state        defs.StateMap
	localRecords map[string][]byte
}

func CreateNewDNSService(c *net.UDPConn, l map[string][]byte) defs.DNSService {
	return &DNSServiceConstruct{
		conn:         c,
		state:        CreateNewStateMap(),
		localRecords: l,
	}
}

func (svc DNSServiceConstruct) Start() {
	//var testm dnsmessage.Message
	// testresp, err := BuildDNSMessage()
	// err2 := testm.Unpack(testresp)
	// if err != nil || err2 != nil {
	// 	log.Printf("fail whale")
	// }
	//log.Printf(testm.Answers[0].Header.Name.String())
	// testhash, err := HashMessageFields(&testm)
	// if err != nil {
	// 	log.Fatalf("error encountered: %v", err)
	// }
	//

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
			key, err := HashMessageFields(&packed)
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
			key, err := HashMessageFields(&packed)
			if err != nil {
				log.Fatalf("error encountered: %v", err)
			}
			if svc.localRecords[key] != nil {
				log.Printf("found a local record for hash: %s", key)
				go svc.conn.WriteToUDP(svc.localRecords[key], addr)
			}

			//testing
			// log.Printf("req key: %s", key)
			// log.Printf("test key: %s", testhash)
			// testm.Header.ID = m.Header.ID
			// res, e := testm.Pack()
			// if e != nil {
			// 	log.Fatalf("could not pack")
			// }
			// if testhash == key {
			// 	go svc.conn.WriteToUDP(res, addr)
			// 	continue
			// }
			//
			svc.state.AddRequestor(key, addr)
			resolver := net.UDPAddr{IP: net.IP{1, 1, 1, 1}, Port: 53}
			go svc.conn.WriteToUDP(packed, &resolver)
		}
	}
}
