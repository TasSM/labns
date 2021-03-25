package main

import (
	"fmt"
	"net"

	"github.com/TasSM/labns/internal/config"
	"github.com/TasSM/labns/internal/logging"
	"github.com/TasSM/labns/internal/service"
)

func main() {
	config.ReadEnvironment()
	go logging.InitLogging(config.LOG_FILE_PATH)
	conf, err := config.LoadConfig(config.CONFIG_FILE_PATH)
	if err != nil {
		logging.LogMessage(logging.LogFatal, "Failed to load configuration file: "+err.Error())
		return
	}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: int(config.SERVICE_DNS_PORT)})
	if err != nil {
		logging.LogMessage(logging.LogFatal, fmt.Sprintf("Failed to bind UDP listener for DNS service on port: %d", config.SERVICE_DNS_PORT))
		return
	}
	service.StartDNSService(conn, conf)
}
