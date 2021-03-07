package dns

import (
	"errors"
	"log"
	"net"

	"github.com/TasSM/labns/internal/config"
	"github.com/TasSM/labns/internal/defs"
	"golang.org/x/net/dns/dnsmessage"
)

func CreateLocalRecords(conf *config.Configuration) (map[string][]byte, error) {
	out := make(map[string][]byte)
	for _, v := range conf.LocalRecords {
		msg, err := BuildDNSMessage(&v)
		if err != nil {
			return nil, err
		}
		hash, err := HashMessageFields(&msg)
		if err != nil {
			return nil, err
		}
		out[hash] = msg
	}
	return out, nil
}

func BuildDNSMessage(record *defs.LocalDNSRecord) ([]byte, error) {
	buf := make([]byte, 2, 514)
	builder := dnsmessage.NewBuilder(buf, dnsmessage.Header{Response: true})
	builder.EnableCompression()
	name, err := dnsmessage.NewName(record.Name)
	if err != nil {
		return nil, err
	}
	err = builder.StartAnswers()
	if err != nil {
		return nil, err
	}
	switch record.Type {
	case "CNAME":
		name, err := dnsmessage.NewName(record.Target)
		if err != nil {
			return nil, err
		}
		res := dnsmessage.CNAMEResource{CNAME: name}
		err = builder.CNAMEResource(dnsmessage.ResourceHeader{Name: name, Class: dnsmessage.ClassINET, TTL: record.TTL}, res)
	case "A":
		ipv4 := [4]byte{}
		ip, err := net.ParseIP(record.Target).MarshalText()
		if err != nil {
			return nil, err
		}
		if ip == nil {
			return nil, errors.New("invalid IPv4 used as target")
		}
		copy(ipv4[:], ip)
		err = builder.AResource(dnsmessage.ResourceHeader{Name: name, Class: dnsmessage.ClassINET, TTL: record.TTL}, dnsmessage.AResource{A: ipv4})
	case "AAAA":
		ipv6 := [16]byte{}
		ip, err := net.ParseIP(record.Target).MarshalText()
		if err != nil {
			return nil, err
		}
		if ip == nil {
			return nil, errors.New("invalid IPv6 used as target")
		}
		copy(ipv6[:], ip)
		err = builder.AAAAResource(dnsmessage.ResourceHeader{Name: name, Class: dnsmessage.ClassINET, TTL: record.TTL}, dnsmessage.AAAAResource{AAAA: ipv6})
	}
	if err != nil {
		log.Printf(err.Error())
	}
	msg, e := builder.Finish()
	if e != nil {
		log.Printf("this shouldnt happen")
		return nil, e
	}
	// var message dnsmessage.Message
	// pe := message.Unpack(msg[2:])
	// if pe != nil {
	// 	log.Printf("error parsing message")
	// }
	// log.Printf("%v", message.Answers[0].Body)
	return msg[2:], nil
}

// func setClassFromType(recordType string) (dnsmessage.Class, error) {
// 	switch recordType {
// 	case "CNAME":
// 		return dnsmessage.ClassINET, nil
// 	case "A":
// 		return dnsmessage.ClassINET, nil
// 	case "AAAA":
// 		return dnsmessage.ClassINET, nil
// 	}
// 	return 0, errors.New("Cannot resolve dnsmessage class from provided record type")
// }

// function to add response ID into the local records
