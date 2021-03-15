package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"

	"github.com/TasSM/labns/internal/defs"
	"github.com/TasSM/labns/internal/logging"
)

const (
	config_path = "./test.json" //"/etc/labdns/config.json"
)

func LoadConfig() (*defs.Configuration, error) {
	file, err := os.Open(config_path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	config := &defs.Configuration{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		return nil, err
	}
	for k, v := range config.LocalRecords {
		if !isValidRecordName(v.Name) {
			return nil, errors.New(fmt.Sprintf("Name for LocalRecord at index %d is invalid, should follow pattern domain.name.:", k))
		}
		if !isValidType(v.Type) {
			return nil, errors.New(fmt.Sprintf("Type for LocalRecord at index %d is invalid:", k))
		}
		if v.TTL == 0 {
			return nil, errors.New(fmt.Sprintf("TTL for LocalRecord at index %d is invalid", k))
		}
		if !isValidTarget(v.Type, v.Target) {
			return nil, errors.New(fmt.Sprintf("Target for LocalRecord at index %d is invalid (check type and target format)", k))
		}
	}
	err = ValidateNameserver(&config.UpstreamNameservers.Primary)
	if err != nil {
		return nil, err
	}
	err = ValidateNameserver(&config.UpstreamNameservers.Secondary)
	if err != nil {
		return nil, err
	}
	if config.UpstreamNameservers.TimeoutMs == 0 {
		config.UpstreamNameservers.TimeoutMs = 5000
	}
	return config, nil
}

func ValidateNameserver(ns *defs.Nameserver) error {
	if ns.Port == 0 {
		ns.Port = 53
	}
	if ns.IPv4 == "" && ns.IPv6 == "" {
		return errors.New(fmt.Sprintf("IPv4 OR IPv6 of upstream nameserver must be provided"))
	}
	if ns.IPv4 != "" {
		parsed := net.ParseIP(ns.IPv4)
		if parsed == nil {
			return errors.New(fmt.Sprintf("IPv4 of upstream nameserver is invalid: %v", ns.IPv4))
		}
	}
	if ns.IPv6 != "" {
		parsed := net.ParseIP(ns.IPv6)
		if parsed == nil {
			return errors.New(fmt.Sprintf("IPv6 of upstream nameserver is invalid %v", ns.IPv6))
		}
	}
	return nil
}

func isValidRecordName(name string) bool {
	matched, err := regexp.MatchString(defs.VALID_FQDN_REGEX, name)
	if err != nil {
		logging.LogMessage(logging.LogFatal, err.Error())
		return false
	}
	return matched
}

func isValidType(parsedType string) bool {
	for _, v := range defs.PermittedRecordTypes {
		if parsedType == v {
			return true
		}
	}
	return false
}

/*
*	Note: Poor approximation of what is actually a valid FQDN for a CNAME records
 */
func isValidTarget(parsedType string, parsedTarget string) bool {
	runes := []rune(parsedTarget)
	switch parsedType {
	case "A":
		return net.ParseIP(parsedTarget).To4() != nil
	case "AAAA":
		return net.ParseIP(parsedTarget).To16() != nil
	case "CNAME":
		matched, err := regexp.MatchString(defs.VALID_FQDN_REGEX, parsedTarget)
		if err != nil {
			logging.LogMessage(logging.LogFatal, err.Error())
			return false
		}
		for i := 0; i < len(runes)-1; i++ {
			if runes[i+1] == runes[i] {
				return false
			}
		}
		return matched
	}
	return false
}
