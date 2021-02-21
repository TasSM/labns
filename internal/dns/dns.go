package dns

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"sort"

	"github.com/TasSM/labns/internal/defs"
	"golang.org/x/net/dns/dnsmessage"
)

type BaseDNSService struct {
	conn           *net.UDPConn
	responseStream chan string
}

func CreateBaseDNSService(c *net.UDPConn) defs.DNSService {
	return &BaseDNSService{
		conn:           c,
		responseStream: make(chan string, 32),
	}
}

func (svc BaseDNSService) hashMessageFields(m *dnsmessage.Message) (string, error) {
	hf := md5.New()
	defer hf.Reset()
	var arr []string
	var sortedString = ""
	if m.Header.Response {
		for _, v := range m.Answers {
			arr = append(arr, v.Header.Name.String())
		}
		sort.Strings(arr)
		for _, v := range arr {
			sortedString += v
		}
		hf.Write([]byte(sortedString))
		return string([]byte(hex.EncodeToString(hf.Sum(nil))[15:31])), nil
	}
	for _, v := range m.Questions {
		arr = append(arr, v.Name.String())
	}
	sort.Strings(arr)
	for _, v := range arr {
		sortedString += v
	}
	hf.Write([]byte(sortedString))
	return string([]byte(hex.EncodeToString(hf.Sum(nil))[15:31])), nil
}

func (svc BaseDNSService) StartListener() {
	log.Printf("Starting Listener service on port %s", svc.conn.LocalAddr().String())
	for {
		buf := make([]byte, 512)
		_, addr, _ := svc.conn.ReadFromUDP(buf)
		var m dnsmessage.Message
		err := m.Unpack(buf)
		if err != nil {
			log.Printf("invalid dns message received from addr %s: $v", addr, buf)
			continue
		}
		packed, _ := m.Pack()
		if m.Header.Response {
			//svc.responseStream <- DNSMessage{source: addr, msg: packed}
			h, err := svc.hashMessageFields(&m)
			if err != nil {
				fmt.Errorf("error encountered: %v", err)
			}
			log.Printf("response hash: %s", h)
			continue
		} else {
			//add m.questions.name into a string and hash it, substring the end, save it with requestors net.UDPAddr
			//check local records
			//check cache
			//forward
			h, err := svc.hashMessageFields(&m)
			if err != nil {
				fmt.Errorf("error encountered: %v", err)
			}
			log.Printf("request hash: %s", h)
			resolver := net.UDPAddr{IP: net.IP{1, 1, 1, 1}, Port: 53}
			go svc.conn.WriteToUDP(packed, &resolver)
		}
	}
}

func (svc BaseDNSService) StartDispatcher() {
	for {
		select {
		case req, ok := <-svc.responseStream:
			if !ok {
				log.Printf("DNSService Dispatcher received close signal")
				return
			}
			go func() {
				_, err := svc.conn.WriteToUDP(req.msg, req.source)
				if err != nil {
					log.Panicf("Could not write DNS response")
					return
				}
			}()
		}
	}
}
