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

var (
	PermittedRecordTypes []string = []string{"A", "AAAA", "CNAME"}
)

type Configuration struct {
	LocalRecords []defs.LocalDNSRecord
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
	return config, nil
}

func isValidRecordName(name string) bool {
	matched, err := regexp.MatchString(`^[a-zA-Z0-9-.]*\.$`, name)
	if err != nil {
		log.Println(err.Error())
		return false
	}
	return matched
}

func isValidType(parsedType string) bool {
	for _, v := range PermittedRecordTypes {
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
		matched, err := regexp.MatchString(`^[a-zA-Z0-9-.]*$`, parsedTarget)
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
