package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"

	"github.com/TasSM/labns/internal/defs"
)

const (
	config_path = "./test.json" //"/etc/labdns/config.json"
)

type Configuration struct {
	LocalRecords        []defs.LocalDNSRecord
	UpstreamNameservers []defs.UpstreamNameserver
}

func LoadConfig() (*Configuration, error) {
	file, err := os.Open(config_path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	config := &Configuration{}
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
	for k, v := range config.UpstreamNameservers {
		if v.Port == 0 {
			v.Port = 53
		}
		if v.IPv4 == "" && v.IPv6 == "" {
			return nil, errors.New(fmt.Sprintf("IPv4 OR IPv6 of upstream nameserver at index %v must be provided", k))
		}
		if v.IPv4 != "" {
			parsed := net.ParseIP(v.IPv4)
			if parsed == nil {
				return nil, errors.New(fmt.Sprintf("IPv4 of upstream nameserver at index %v is invalid", k))
			}
		}
		if v.IPv6 != "" {
			parsed := net.ParseIP(v.IPv4)
			if parsed == nil {
				return nil, errors.New(fmt.Sprintf("IPv6 of upstream nameserver at index %v is invalid", k))
			}
		}
	}
	return config, nil
}

func isValidRecordName(name string) bool {
	matched, err := regexp.MatchString(defs.VALID_FQDN_REGEX, name)
	if err != nil {
		log.Println(err.Error())
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
			log.Println(err.Error())
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
