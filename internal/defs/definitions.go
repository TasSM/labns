package defs

import "net"

type DNSService interface {
	Start()
}

type StateMap interface {
	AddRequestor(key string, addr *net.UDPAddr)
	RetrieveAndDelete(key string) ([]*net.UDPAddr, error)
}
