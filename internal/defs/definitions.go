package defs

import "net"

type DNSService interface {
	Start()
}

type StateMap interface {
	AddRequestor(key string, addr *net.UDPAddr)
	RetrieveAndDelete(key string) ([]*net.UDPAddr, error)
}

type LocalDNSRecord struct {
	Name   string
	Type   string
	TTL    uint32
	Target string
}

type PendingRequestState struct {
	Addr      net.Addr
	RequestId string
}
