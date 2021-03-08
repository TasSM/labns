package service

import (
	"crypto/md5"
	"encoding/hex"
	"sort"

	"golang.org/x/net/dns/dnsmessage"
)

func HashMessageFields(msgSerial *[]byte) (string, error) {
	hf := md5.New()
	defer hf.Reset()
	var m dnsmessage.Message
	var arr []string
	var sortedString = ""
	err := m.Unpack(*msgSerial)
	if err != nil {
		return "", err
	}
	for _, v := range m.Questions {
		arr = append(arr, v.Name.String())
		arr = append(arr, v.Type.String())
	}
	sort.Strings(arr)
	for _, v := range arr {
		sortedString += v
	}
	hf.Write([]byte(sortedString))
	return string([]byte(hex.EncodeToString(hf.Sum(nil))[15:31])), nil
}
