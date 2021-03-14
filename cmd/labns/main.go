package main

import (
	"log"
	"net"

	"github.com/TasSM/labns/internal/config"
	"github.com/TasSM/labns/internal/defs"
	"github.com/TasSM/labns/internal/service"
)

func main() {
	conf, err := config.LoadConfig()
	log.Printf("%v", conf)
	if err != nil {
		log.Fatalf("failed to load configuration %s", err.Error())
	}

	conn, _ := net.ListenUDP("udp", &net.UDPAddr{Port: defs.SERVICE_DNS_PORT})
	service.StartDNSService(conn, conf)
}
