package defs

import (
	"net"

	"golang.org/x/net/dns/dnsmessage"
)

const (
	SERVICE_DNS_PORT = 2001
	VALID_FQDN_REGEX = `^[a-zA-Z0-9-.]*\.$`
)

var (
	RecordTypeMap = map[string]dnsmessage.Type{
		"CNAME": dnsmessage.TypeCNAME,
		"AAAA":  dnsmessage.TypeAAAA,
		"A":     dnsmessage.TypeA,
	}
	PermittedRecordTypes []string = []string{"A", "AAAA", "CNAME"}
)

type DNSService interface {
	Start()
}

type PendingRequestState struct {
	Addr      *net.UDPAddr
	RequestId uint16
}

type StateMap interface {
	AddRequestor(key string, addr *net.UDPAddr, id uint16) bool
	RetrieveAndDelete(key string) ([]PendingRequestState, error)
}

type LocalDNSRecord struct {
	Name   string
	Type   string
	TTL    uint32
	Target string
}

type UpstreamNameserver struct {
	IPv4 string
	IPv6 string
	Port uint16
}
