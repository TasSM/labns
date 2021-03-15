package service

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/TasSM/labns/internal/defs"
	"github.com/TasSM/labns/internal/logging"
	"golang.org/x/net/dns/dnsmessage"
)

var (
	conn     *net.UDPConn
	stateMap map[string][]defs.PendingRequestState
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
		logging.LogMessage(logging.LogFatal, err.Error())
	}
	go conn.WriteToUDP(res, target.Addr)
}

func switchNameservers(conf *defs.Configuration) {
	tmp := conf.UpstreamNameservers.Primary
	conf.UpstreamNameservers.Primary = conf.UpstreamNameservers.Secondary
	conf.UpstreamNameservers.Secondary = tmp
}

func startStateWorker(input chan defs.StateOperation, conf *defs.Configuration) {
	locConf := *conf
	stateMap = make(map[string][]defs.PendingRequestState)
	localRecords, err := CreateLocalRecords(&locConf)
	if err != nil {
		logging.LogMessage(logging.LogFatal, "Failed to create local record: "+err.Error())
	}
	for {
		select {
		case op, ok := <-input:
			if !ok {
				logging.LogMessage(logging.LogFatal, "Command channel closed, killing state worker")
				return
			}
			if op.Operation == 0 || op.RequestKey == "" {
				logging.LogMessage(logging.LogError, "Received invalid state operation, continuing...")
				continue
			}
			switch op.Operation {
			case defs.OpAdd:
				if op.RequestData == nil || op.ByteData == nil || op.RequestKey == "" {
					logging.LogMessage(logging.LogError, "Bad OpAdd (missing required data), continuing...")
					continue
				}
				if localRecords[op.RequestKey] != nil {
					logging.LogMessage(logging.LogInfo, "Found local record with matching key: "+op.RequestKey)
					go writeResponse(localRecords[op.RequestKey], op.RequestData)
					continue
				}
				if stateMap[op.RequestKey] == nil {
					stateMap[op.RequestKey] = make([]defs.PendingRequestState, 16)
				}
				stateMap[op.RequestKey] = append(stateMap[op.RequestKey], *op.RequestData)
				err := requestUpstream(&locConf.UpstreamNameservers.Primary, op.ByteData)
				if err != nil {
					logging.LogMessage(logging.LogError, "Unable to forward request to upstream: "+err.Error())
				}
				go func() {
					time.Sleep(time.Duration(locConf.UpstreamNameservers.TimeoutMs) * time.Millisecond)
					input <- defs.StateOperation{Operation: defs.OpCallback, RequestKey: op.RequestKey, ByteData: op.ByteData}
				}()
			case defs.OpCallback:
				if stateMap[op.RequestKey] == nil || len(stateMap[op.RequestKey]) == 0 {
					continue
				}
				if op.ByteData == nil || op.RequestKey == "" {
					logging.LogMessage(logging.LogError, "Bad OpCallback (missing required data), continuing...")
					continue
				}
				err := requestUpstream(&locConf.UpstreamNameservers.Secondary, op.ByteData)
				if err != nil {
					logging.LogMessage(logging.LogError, "Unable to forward request to upstream: "+err.Error())
				}
				switchNameservers(&locConf)
				go func() {
					time.Sleep(time.Duration(locConf.UpstreamNameservers.TimeoutMs) * time.Millisecond)
					input <- defs.StateOperation{Operation: defs.OpDelete, RequestKey: op.RequestKey}
				}()
			case defs.OpRespond:
				if op.Conn == nil || op.ByteData == nil || op.RequestKey == "" {
					logging.LogMessage(logging.LogError, "Bad OpRespond (missing required data), continuing...")
					continue
				}
				if stateMap[op.RequestKey] == nil || len(stateMap[op.RequestKey]) == 0 {
					logging.LogMessage(logging.LogDebug, "OpRespond ignored for missing key "+op.RequestKey)
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
					continue
				}
				logging.LogMessage(logging.LogError, "Request for key "+op.RequestKey+" has timed out on both upstream nameservers")
				switchNameservers(&locConf)
				delete(stateMap, op.RequestKey)
			}
		}
	}
}

func StartDNSService(c *net.UDPConn, conf *defs.Configuration) {
	conn = c
	reqChan := make(chan defs.StateOperation, 64)
	go startStateWorker(reqChan, conf)
	logging.LogMessage(logging.LogInfo, "Starting Listener service on port "+conn.LocalAddr().String())
	for {
		buf := make([]byte, 512)
		_, addr, _ := conn.ReadFromUDP(buf)
		var m dnsmessage.Message
		err := m.Unpack(buf)
		if err != nil {
			logging.LogMessage(logging.LogError, fmt.Sprintf("Invalid DNS message received from %v - skipping", addr))
			continue
		}
		packed, _ := m.Pack()
		key, err := HashMessageFields(&packed)
		if m.Header.Response {
			logging.LogMessage(logging.LogInfo, fmt.Sprintf("Received resource response from upstream %s for %s: %s", conf.UpstreamNameservers.Primary.IPv4, m.Questions[0].Name, GetAddressFromResource(m.Answers[0])))
			if err != nil {
				logging.LogMessage(logging.LogFatal, err.Error())
			}
			reqChan <- defs.StateOperation{Operation: defs.OpRespond, RequestKey: key, ByteData: packed, Conn: conn}
			continue
		} else {
			logging.LogMessage(logging.LogInfo, fmt.Sprintf("Received resource request for %v", m.Questions[0].Name))
			if err != nil {
				logging.LogMessage(logging.LogFatal, err.Error())
			}
			reqChan <- defs.StateOperation{Operation: defs.OpAdd, RequestKey: key, RequestData: &defs.PendingRequestState{Addr: addr, RequestId: m.ID}, ByteData: packed}
		}
	}
}
