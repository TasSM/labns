package service

import (
	"log"
	"net"

	"golang.org/x/net/dns/dnsmessage"
)

var conn *net.UDPConn
var localRecords map[string][]byte
var state = CreateNewStateMap()

// move state management to a goroutine
// var stateWriter chan int
// var stateReader chan int

func StartDNSService(c *net.UDPConn, l map[string][]byte) {
	conn = c
	localRecords = l
	log.Printf("Starting Listener service on port %s", conn.LocalAddr().String())
	for {
		buf := make([]byte, 512)
		_, addr, _ := conn.ReadFromUDP(buf)
		var m dnsmessage.Message
		err := m.Unpack(buf)
		if err != nil {
			log.Printf("Invalid DNS message received from addr %s: $v", addr, buf)
			continue
		}
		packed, _ := m.Pack()
		if m.Header.Response {
			log.Printf("Received response for %v: %v", m.Questions, m.Answers)
			key, err := HashMessageFields(&packed)
			if err != nil {
				log.Fatalf("error encountered: %v", err.Error())
			}
			go func() {
				log.Printf("key is %v", key)
				adlist, err := state.RetrieveAndDelete(key)
				if err != nil {
					log.Fatalf("Received invalid key from upstream request: %v", err.Error())
				}
				if len(adlist) == 1 {
					go conn.WriteToUDP(packed, adlist[0].Addr)
					return
				}
				for _, v := range adlist {
					res, err := SetResponseId(packed, v.RequestId)
					if err != nil {
						log.Fatalf("Unable to set response ID: %v", err.Error())
					}
					go conn.WriteToUDP(res, v.Addr)
				}
			}()
			continue
		} else {
			log.Printf("Received request: %v", m.Questions)
			key, err := HashMessageFields(&packed)
			if err != nil {
				log.Fatalf("error encountered: %v", err.Error())
			}
			if localRecords[key] != nil {
				log.Printf("Found a local record for message hash: %s", key)
				res, err := SetResponseId(localRecords[key], m.ID)
				if err != nil {
					log.Fatalf("error encountered %v", err.Error())
				}
				go conn.WriteToUDP(res, addr)
				continue
			}
			isCreated := state.AddRequestor(key, addr, m.ID)
			if isCreated {
				resolver := net.UDPAddr{IP: net.IP{1, 1, 1, 1}, Port: 53}
				go conn.WriteToUDP(packed, &resolver)
			}
		}
	}
}
