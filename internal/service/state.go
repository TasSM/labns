package service

import (
	"errors"
	"net"
	"sync"

	"github.com/TasSM/labns/internal/defs"
)

type StateMapConstruct struct {
	PendingRequests map[string][]defs.PendingRequestState
	m               sync.Mutex
}

func CreateNewStateMap() defs.StateMap {
	return &StateMapConstruct{
		PendingRequests: make(map[string][]defs.PendingRequestState),
		m:               sync.Mutex{},
	}
}

func (sm *StateMapConstruct) AddRequestor(key string, addr *net.UDPAddr, id uint16) bool {
	sm.m.Lock()
	defer sm.m.Unlock()
	res := false
	if sm.PendingRequests[key] == nil {
		sm.PendingRequests[key] = make([]defs.PendingRequestState, 16)
		res = true
	}
	sm.PendingRequests[key] = append(sm.PendingRequests[key], defs.PendingRequestState{Addr: addr, RequestId: id})
	return res
}

func (sm *StateMapConstruct) PeekRequestors(key string) []defs.PendingRequestState {
	sm.m.Lock()
	defer sm.m.Unlock()
	return sm.PendingRequests[key]
}

func (sm *StateMapConstruct) RetrieveAndDelete(key string) ([]defs.PendingRequestState, error) {
	sm.m.Lock()
	defer sm.m.Unlock()
	if sm.PendingRequests[key] == nil {
		return nil, errors.New("No entries matching key: " + key)
	}
	defer delete(sm.PendingRequests, key)
	return sm.PendingRequests[key], nil
}
