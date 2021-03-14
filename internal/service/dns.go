package service

import (
	"errors"
	"log"
	"net"

	"github.com/TasSM/labns/internal/defs"
	"golang.org/x/net/dns/dnsmessage"
)

var (
	conn         *net.UDPConn
	localRecords map[string][]byte
)

func requestUpstream(ns *defs.Nameserver, payload []byte) error {
	var target net.UDPAddr
	if ns.IPv4 == "" && ns.IPv6 == "" {
		return errors.New("Cannot forward to invalid upstream: neither IPv4 or IPv6 specified")
	}
	if ns.IPv4 != "" {
		ipv4 := [4]byte{}
		ip := net.ParseIP(ns.IPv4).To4()
		copy(ipv4[:], ip)
		target = net.UDPAddr{IP: ip, Port: int(ns.Port)}
	} else {
		ipv6 := [16]byte{}
		ip := net.ParseIP(ns.IPv6).To16()
		copy(ipv6[:], ip)
		target = net.UDPAddr{IP: ip, Port: int(ns.Port)}
	}
	go conn.WriteToUDP(payload, &target)
	return nil
}

func StartDNSService(c *net.UDPConn, conf *defs.Configuration) {
	localRecords, err := CreateLocalRecords(conf)
	if err != nil {
		log.Fatalf("failed to create local records %s", err.Error())
	}
	conn = c
	var reqChan = make(chan defs.StateOperation, 64)
	go processStateFlow(reqChan, conf)
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
			//trigger response writing
			reqChan <- defs.StateOperation{Operation: defs.OpRespond, RequestKey: key, ByteData: packed, Conn: conn}
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
			//trigger a new request state to manage
			reqChan <- defs.StateOperation{Operation: defs.OpAdd, RequestKey: key, RequestData: &defs.PendingRequestState{Addr: addr, RequestId: m.ID}, ByteData: packed}
		}
	}
}
