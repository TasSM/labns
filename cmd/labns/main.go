package main

import (
	"net"

	"github.com/TasSM/labns/internal/dns"
)

const (
	SERVICE_DNS_PORT = 2001
)

func main() {
	conn, _ := net.ListenUDP("udp", &net.UDPAddr{Port: SERVICE_DNS_PORT})
	service := dns.CreateBaseDNSService(conn)
	service.StartListener()
}
