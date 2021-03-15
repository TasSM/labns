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

type Operation uint16

const (
	OpCallback Operation = 1
	OpAdd      Operation = 2
	OpRespond  Operation = 3
	OpDelete   Operation = 4
)

type PendingRequestState struct {
	Addr      *net.UDPAddr
	RequestId uint16
}

type LocalDNSRecord struct {
	Name   string
	Type   string
	TTL    uint32
	Target string
}

type Nameserver struct {
	IPv4 string
	IPv6 string
	Port uint16
}

type UpstreamNameservers struct {
	Primary   Nameserver
	Secondary Nameserver
	TimeoutMs uint16
}

type Configuration struct {
	LocalRecords        []LocalDNSRecord
	UpstreamNameservers UpstreamNameservers
}

type StateOperation struct {
	Operation   Operation
	RequestKey  string
	RequestData *PendingRequestState
	ByteData    []byte
	Conn        *net.UDPConn
}
