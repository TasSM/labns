package main

import (
	"net"

	"github.com/TasSM/labns/internal/config"
	"github.com/TasSM/labns/internal/defs"
	"github.com/TasSM/labns/internal/logging"
	"github.com/TasSM/labns/internal/service"
)

func main() {
	go logging.InitLogging("labns.log")
	conf, err := config.LoadConfig()
	if err != nil {
		logging.LogMessage(logging.LogFatal, "Failed to load configuration file: "+err.Error())
	}

	conn, _ := net.ListenUDP("udp", &net.UDPAddr{Port: defs.SERVICE_DNS_PORT})
	service.StartDNSService(conn, conf)
}
