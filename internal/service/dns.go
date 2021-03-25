package service

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/TasSM/labns/internal/config"
	"github.com/TasSM/labns/internal/logging"
	"golang.org/x/net/dns/dnsmessage"
)

type Operation uint16

type StateOperation struct {
	Operation     Operation
	RequestHash   string
	RequestorAddr *net.UDPAddr
	ByteData      []byte
	RequestId     uint16
}

const (
	OpCallback Operation = 1
	OpAdd      Operation = 2
	OpRespond  Operation = 3
	OpDelete   Operation = 4
)

var (
	lock            sync.Mutex
	conn            *net.UDPConn
	stateMap        map[uint16]*net.UDPAddr
	currentUpstream string
)

func requestUpstream(ns *config.Nameserver, payload []byte) error {
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

func switchNameservers(conf *config.Configuration) {
	lock.Lock()
	defer lock.Unlock()
	tmp := conf.UpstreamNameservers.Primary
	currentUpstream = conf.UpstreamNameservers.Secondary.IPv4
	conf.UpstreamNameservers.Primary = conf.UpstreamNameservers.Secondary
	conf.UpstreamNameservers.Secondary = tmp
}

func startStateWorker(input chan StateOperation, conf *config.Configuration) {
	locConf := *conf
	currentUpstream = locConf.UpstreamNameservers.Primary.IPv4
	stateMap = make(map[uint16]*net.UDPAddr)
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
			if op.Operation == 0 || (op.RequestHash == "" && op.RequestId == 0) {
				logging.LogMessage(logging.LogError, "Received invalid state operation, continuing...")
				continue
			}
			switch op.Operation {
			case OpAdd:
				if op.RequestorAddr == nil || op.ByteData == nil || op.RequestHash == "" || op.RequestId == 0 {
					logging.LogMessage(logging.LogError, "Bad OpAdd (missing required data), continuing...")
					continue
				}
				if localRecords[op.RequestHash] != nil {
					logging.LogMessage(logging.LogInfo, "Found local record with matching key: "+op.RequestHash)
					res, err := SetResponseId(localRecords[op.RequestHash], op.RequestId)
					if err != nil {
						logging.LogMessage(logging.LogFatal, err.Error())
						continue
					}
					go conn.WriteToUDP(res, op.RequestorAddr)
					continue
				}
				//check cache here
				stateMap[op.RequestId] = op.RequestorAddr
				err := requestUpstream(&locConf.UpstreamNameservers.Primary, op.ByteData)
				if err != nil {
					logging.LogMessage(logging.LogError, "Unable to forward request to upstream: "+err.Error())
				}
				go func() {
					time.Sleep(time.Duration(locConf.UpstreamNameservers.TimeoutMs) * time.Millisecond)
					input <- StateOperation{Operation: OpCallback, ByteData: op.ByteData, RequestId: op.RequestId, RequestorAddr: op.RequestorAddr}
				}()
			case OpCallback:
				if op.ByteData == nil || op.RequestorAddr == nil || op.RequestId == 0 {
					logging.LogMessage(logging.LogError, "Bad OpCallback (missing required data), continuing...")
					continue
				}
				if stateMap[op.RequestId] == nil {
					continue
				}
				err := requestUpstream(&locConf.UpstreamNameservers.Secondary, op.ByteData)
				if err != nil {
					logging.LogMessage(logging.LogError, "Unable to forward request to upstream: "+err.Error())
				}
				logging.LogMessage(logging.LogInfo, "Primary upstream timed out, switching primary ("+locConf.UpstreamNameservers.Primary.IPv4+") and secondary ("+locConf.UpstreamNameservers.Secondary.IPv4+")")
				switchNameservers(&locConf)
				go func() {
					time.Sleep(time.Duration(locConf.UpstreamNameservers.TimeoutMs) * time.Millisecond)
					input <- StateOperation{Operation: OpDelete, RequestId: op.RequestId}
				}()
			case OpRespond:
				if op.ByteData == nil || op.RequestId == 0 {
					logging.LogMessage(logging.LogError, "Bad OpRespond (missing required data), continuing...")
					continue
				}
				if stateMap[op.RequestId] == nil {
					logging.LogMessage(logging.LogDebug, "OpRespond ignored for missing key "+op.RequestHash)
					continue
				}
				go conn.WriteToUDP(op.ByteData, stateMap[op.RequestId])
				delete(stateMap, op.RequestId)
			case OpDelete:
				if op.RequestId == 0 {
					logging.LogMessage(logging.LogError, "Bad OpDelete (missing required data), continuing...")
					continue
				}
				if stateMap[op.RequestId] == nil {
					continue
				}
				logging.LogMessage(logging.LogError, "Request for key "+op.RequestHash+" has timed out on both upstream nameservers")
				switchNameservers(&locConf)
				delete(stateMap, op.RequestId)
			}
		}
	}
}

func StartDNSService(c *net.UDPConn, conf *config.Configuration) {
	conn = c
	reqChan := make(chan StateOperation, 64)
	var logMsg string
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
		if err != nil {
			logging.LogMessage(logging.LogFatal, err.Error())
		}
		if m.Header.Response {
			lock.Lock()
			logMsg = fmt.Sprintf("Received %s response from upstream %s for %s", m.Questions[0].Type, currentUpstream, m.Questions[0].Name)
			lock.Unlock()
			if len(m.Answers) > 0 {
				logMsg = logMsg + GetAddressFromResource(m.Answers[0])
			} else {
				logMsg = logMsg + ": empty "
			}
			logging.LogMessage(logging.LogInfo, logMsg)
			reqChan <- StateOperation{Operation: OpRespond, RequestId: m.ID, ByteData: packed}
			continue
		} else {
			if len(m.Questions) == 0 {
				continue
			}
			logging.LogMessage(logging.LogInfo, fmt.Sprintf("Received resource request for %v", m.Questions[0].Name))
			reqChan <- StateOperation{Operation: OpAdd, RequestHash: key, RequestorAddr: addr, RequestId: m.ID, ByteData: packed}
		}
	}
}
