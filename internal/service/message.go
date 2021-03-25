package service

import (
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/TasSM/labns/internal/config"
	"github.com/TasSM/labns/internal/logging"
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

func GetAddressFromResource(resource dnsmessage.Resource) string {
	str := resource.Body.GoString()
	res := ""
	switch resource.Header.Type {
	case dnsmessage.TypeA:
		res = strings.ReplaceAll(str[strings.LastIndex(str, "{")+1:strings.Index(str, "}")], ", ", ".")
	case dnsmessage.TypeAAAA:
		tmp := strings.Split(strings.ReplaceAll(str[strings.LastIndex(str, "{")+1:strings.Index(str, "}")], ",", ""), " ")
		bytes := make([]byte, 16)
		for i := 0; i < 16; i++ {
			val, err := strconv.ParseUint(tmp[i], 10, 8)
			if err != nil {
				logging.LogMessage(logging.LogError, "Unable to parse byte value from string "+tmp[i])
			}
			bytes[i] = uint8(val)
		}
		var ip net.IP = bytes
		res = ip.To16().String()
	}
	return res
}

func BuildDNSMessage(record *config.LocalDNSRecord) ([]byte, error) {
	buf := make([]byte, 2, 514)
	builder := dnsmessage.NewBuilder(buf, dnsmessage.Header{Response: true})
	builder.EnableCompression()
	name, err := dnsmessage.NewName(record.Name)
	recordType := config.RecordTypeMap[record.Type]
	if recordType == 0 {
		return nil, errors.New("local records question type was not set to a valid value")
	}
	question := dnsmessage.Question{Name: name, Type: recordType, Class: dnsmessage.ClassINET}
	header := dnsmessage.ResourceHeader{Name: name, Class: dnsmessage.ClassINET, TTL: record.TTL}
	if err != nil {
		return nil, err
	}
	err = builder.StartQuestions()
	if err != nil {
		return nil, err
	}
	err = builder.Question(question)
	if err != nil {
		return nil, err
	}
	err = builder.StartAnswers()
	if err != nil {
		return nil, err
	}
	switch record.Type {
	case "CNAME":
		fqdn, err := dnsmessage.NewName(record.Target)
		if err != nil {
			return nil, err
		}
		targRes := dnsmessage.CNAMEResource{CNAME: fqdn}
		err = builder.CNAMEResource(header, targRes)
		if err != nil {
			return nil, err
		}
	case "A":
		ipv4 := [4]byte{}
		ip := net.ParseIP(record.Target).To4()
		if ip == nil {
			return nil, errors.New("invalid IPv4 used as target")
		}
		copy(ipv4[:], ip)
		err = builder.AResource(header, dnsmessage.AResource{A: ipv4})
	case "AAAA":
		ipv6 := [16]byte{}
		ip := net.ParseIP(record.Target).To16()
		if ip == nil {
			return nil, errors.New("invalid IPv6 used as target")
		}
		copy(ipv6[:], ip)
		err = builder.AAAAResource(header, dnsmessage.AAAAResource{AAAA: ipv6})
	}
	if err != nil {
		logging.LogMessage(logging.LogError, err.Error())
	}
	msg, err := builder.Finish()
	if err != nil {
		return nil, err
	}
	return msg[2:], nil
}

func SetResponseId(serial []byte, Id uint16) ([]byte, error) {
	var m dnsmessage.Message
	err := m.Unpack(serial)
	if err != nil {
		return nil, err
	}
	m.ID = Id
	return m.Pack()
}
