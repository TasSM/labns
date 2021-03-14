package service

import (
	"log"
	"time"

	"github.com/TasSM/labns/internal/defs"
)

var (
	stateMap map[string][]defs.PendingRequestState
)

func processStateFlow(input chan defs.StateOperation, conf *defs.Configuration) {
	stateMap = make(map[string][]defs.PendingRequestState)
	for {
		select {
		//check for a close
		case op := <-input:
			if op.Operation == 0 || op.RequestKey == "" {
				log.Printf("bad operation - machine broke")
				continue
			}
			switch op.Operation {
			case defs.OpAdd:
				if op.RequestData == nil || op.ByteData == nil || op.RequestKey == "" {
					log.Printf("bad add operation")
					continue
				}
				if stateMap[op.RequestKey] == nil {
					stateMap[op.RequestKey] = make([]defs.PendingRequestState, 16)
				}
				stateMap[op.RequestKey] = append(stateMap[op.RequestKey], *op.RequestData)
				requestUpstream(&conf.UpstreamNameservers.Primary, op.ByteData)
				go func() {
					time.Sleep(time.Duration(conf.UpstreamNameservers.TimeoutMs) * time.Millisecond)
					input <- defs.StateOperation{Operation: defs.OpCallback, RequestKey: op.RequestKey, ByteData: op.ByteData}
				}()
			case defs.OpCallback:
				if stateMap[op.RequestKey] == nil || len(stateMap[op.RequestKey]) == 0 {
					log.Printf("OpCallback was ignored as response has been processed: %v", op.RequestKey)
					continue
				}
				if op.ByteData == nil || op.RequestKey == "" {
					log.Printf("bad data for ByetData request")
					continue
				}
				requestUpstream(&conf.UpstreamNameservers.Secondary, op.ByteData)
				tmp := conf.UpstreamNameservers.Primary
				conf.UpstreamNameservers.Primary = conf.UpstreamNameservers.Secondary
				conf.UpstreamNameservers.Secondary = tmp
				go func() {
					time.Sleep(time.Duration(conf.UpstreamNameservers.TimeoutMs) * time.Millisecond)
					input <- defs.StateOperation{Operation: defs.OpDelete, RequestKey: op.RequestKey}
				}()
			case defs.OpRespond:
				if op.Conn == nil || op.ByteData == nil || op.RequestKey == "" {
					log.Printf("invalid data in OpRespond")
					continue
				}
				if stateMap[op.RequestKey] == nil || len(stateMap[op.RequestKey]) == 0 {
					log.Printf("OpRespond was ignored as response has been processed: %v", op.RequestKey)
					continue
				}
				if len(stateMap[op.RequestKey]) == 1 {
					go conn.WriteToUDP(op.ByteData, op.RequestData.Addr)
				}
				for _, v := range stateMap[op.RequestKey] {
					res, err := SetResponseId(op.ByteData, v.RequestId)
					if err != nil {
						log.Fatalf("Unable to set response ID: %v", err.Error())
					}
					go conn.WriteToUDP(res, v.Addr)
				}
				delete(stateMap, op.RequestKey)
			case defs.OpDelete:
				if stateMap[op.RequestKey] == nil || len(stateMap[op.RequestKey]) == 0 {
					log.Printf("OpDelete was ignored as response has been processed: %v", op.RequestKey)
					continue
				}
				log.Printf("Request has timed out on both nameservers")
				delete(stateMap, op.RequestKey)
			}
		}
	}
}
