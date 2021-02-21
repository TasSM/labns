package dns

import (
	"crypto/md5"
	"encoding/hex"
	"sort"

	"golang.org/x/net/dns/dnsmessage"
)

func HashMessageFields(m *dnsmessage.Message) (string, error) {
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
