package config

import (
	"os"
	"strconv"
)

const (
	VALID_FQDN_REGEX     = `^[a-zA-Z0-9-.]*\.$`
	ENV_CONFIG_PATH      = "LABNS_CONFIG_PATH"
	ENV_LOG_PATH         = "LABNS_LOG_PATH"
	ENV_DNS_SERVICE_PORT = "LABNS_DNS_SERVICE_PORT"
)

var (
	CONFIG_FILE_PATH string
	LOG_FILE_PATH    string
	SERVICE_DNS_PORT uint16
)

func GetEnv(value string, def string) string {
	check := os.Getenv(value)
	if check != "" {
		return check
	}
	return def
}

func ReadEnvironment() error {
	CONFIG_FILE_PATH = GetEnv(ENV_CONFIG_PATH, "/etc/labns/labns.json")
	LOG_FILE_PATH = GetEnv(ENV_LOG_PATH, "")
	port, err := strconv.ParseUint(GetEnv(ENV_DNS_SERVICE_PORT, "53"), 10, 16)
	if err != nil {
		return err
	}
	SERVICE_DNS_PORT = uint16(port)
	return nil
}
