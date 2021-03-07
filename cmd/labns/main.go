package main

import (
	"log"
	"net"

	"github.com/TasSM/labns/internal/config"
	"github.com/TasSM/labns/internal/dns"
)

const (
	SERVICE_DNS_PORT = 2001
)

func main() {
	conf, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration %s", err.Error())
	}
	locals, err := dns.CreateLocalRecords(conf)
	if err != nil {
		log.Fatalf("failed to create local records %s", err.Error())
	}

	conn, _ := net.ListenUDP("udp", &net.UDPAddr{Port: SERVICE_DNS_PORT})
	service := dns.CreateNewDNSService(conn, locals)
	service.Start()
}
