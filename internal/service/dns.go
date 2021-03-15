package service

import (
	"errors"
	"log"
	"net"
	"time"

	"github.com/TasSM/labns/internal/defs"
	"golang.org/x/net/dns/dnsmessage"
)

var (
	conn         *net.UDPConn
	localRecords map[string][]byte
	stateMap     map[string][]defs.PendingRequestState
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

func writeResponse(res []byte, target *defs.PendingRequestState) {
	res, err := SetResponseId(res, target.RequestId)
	if err != nil {
		log.Fatalf("error encountered %v", err.Error())
	}
	go conn.WriteToUDP(res, target.Addr)
}

func switchNameservers(conf *defs.Configuration) {
	tmp := conf.UpstreamNameservers.Primary
	conf.UpstreamNameservers.Primary = conf.UpstreamNameservers.Secondary
	conf.UpstreamNameservers.Secondary = tmp
}

func startStateWorker(input chan defs.StateOperation, conf *defs.Configuration) {
	stateMap = make(map[string][]defs.PendingRequestState)
	localRecords, err := CreateLocalRecords(conf)
	if err != nil {
		log.Fatalf("failed to create local records %s", err.Error())
	}
	for {
		select {
		//check for a close
		case op, ok := <-input:
			if !ok {
				log.Fatalf("Command channel closed - killing state worker")
				return
			}
			if op.Operation == 0 || op.RequestKey == "" {
				log.Printf("bad operation - machine broke")
				continue
			}
			switch op.Operation {
			case defs.OpAdd:
				if op.RequestData == nil || op.ByteData == nil || op.RequestKey == "" {
					log.Printf("bad add operation")
					continue
				}
				if localRecords[op.RequestKey] != nil {
					log.Printf("Found a local record for message hash: %s", op.RequestKey)
					go writeResponse(localRecords[op.RequestKey], op.RequestData)
					continue
				}
				if stateMap[op.RequestKey] == nil {
					stateMap[op.RequestKey] = make([]defs.PendingRequestState, 16)
				}
				stateMap[op.RequestKey] = append(stateMap[op.RequestKey], *op.RequestData)
				requestUpstream(&conf.UpstreamNameservers.Primary, op.ByteData)
				go func() {
					time.Sleep(time.Duration(conf.UpstreamNameservers.TimeoutMs) * time.Millisecond)
					input <- defs.StateOperation{Operation: defs.OpCallback, RequestKey: op.RequestKey, ByteData: op.ByteData}
				}()
			case defs.OpCallback:
				if stateMap[op.RequestKey] == nil || len(stateMap[op.RequestKey]) == 0 {
					log.Printf("OpCallback was ignored as response has been processed: %v", op.RequestKey)
					continue
				}
				if op.ByteData == nil || op.RequestKey == "" {
					log.Printf("bad data for ByteData request")
					continue
				}
				requestUpstream(&conf.UpstreamNameservers.Secondary, op.ByteData)
				switchNameservers(conf)
				go func() {
					time.Sleep(time.Duration(conf.UpstreamNameservers.TimeoutMs) * time.Millisecond)
					input <- defs.StateOperation{Operation: defs.OpDelete, RequestKey: op.RequestKey}
				}()
			case defs.OpRespond:
				if op.Conn == nil || op.ByteData == nil || op.RequestKey == "" {
					log.Printf("invalid data in OpRespond")
					continue
				}
				if stateMap[op.RequestKey] == nil || len(stateMap[op.RequestKey]) == 0 {
					log.Printf("OpRespond was ignored as response has been processed: %v", op.RequestKey)
					continue
				}
				if len(stateMap[op.RequestKey]) == 1 {
					go conn.WriteToUDP(op.ByteData, op.RequestData.Addr)
				}
				for _, v := range stateMap[op.RequestKey] {
					writeResponse(op.ByteData, &v)
				}
				delete(stateMap, op.RequestKey)
			case defs.OpDelete:
				if stateMap[op.RequestKey] == nil || len(stateMap[op.RequestKey]) == 0 {
					log.Printf("OpDelete was ignored as response has been processed: %v", op.RequestKey)
					continue
				}
				log.Printf("Request has timed out on both nameservers")
				switchNameservers(conf)
				delete(stateMap, op.RequestKey)
			}
		}
	}
}

func StartDNSService(c *net.UDPConn, conf *defs.Configuration) {
	conn = c
	var reqChan = make(chan defs.StateOperation, 64)
	go startStateWorker(reqChan, conf)
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
		key, err := HashMessageFields(&packed)
		if m.Header.Response {
			log.Printf("Received response for %v: %v", m.Questions, m.Answers)
			if err != nil {
				log.Fatalf("error encountered: %v", err.Error())
			}
			//trigger response writing
			reqChan <- defs.StateOperation{Operation: defs.OpRespond, RequestKey: key, ByteData: packed, Conn: conn}
			continue
		} else {
			log.Printf("Received request: %v", m.Questions)
			if err != nil {
				log.Fatalf("error encountered: %v", err.Error())
			}
			reqChan <- defs.StateOperation{Operation: defs.OpAdd, RequestKey: key, RequestData: &defs.PendingRequestState{Addr: addr, RequestId: m.ID}, ByteData: packed}
		}
	}
}
