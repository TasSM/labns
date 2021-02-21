package dns

import (
	"errors"
	"net"
	"sync"

	"github.com/TasSM/labns/internal/defs"
)

type StateMapConstruct struct {
	PendingRequests map[string][]*net.UDPAddr
	m               sync.Mutex
}

func CreateNewStateMap() defs.StateMap {
	return &StateMapConstruct{
		PendingRequests: make(map[string][]*net.UDPAddr),
		m:               sync.Mutex{},
	}
}

func (sm *StateMapConstruct) AddRequestor(key string, addr *net.UDPAddr) {
	sm.m.Lock()
	defer sm.m.Unlock()
	if sm.PendingRequests[key] == nil {
		sm.PendingRequests[key] = []*net.UDPAddr{addr}
		return
	}
	sm.PendingRequests[key] = append(sm.PendingRequests[key], addr)
}

func (sm *StateMapConstruct) RetrieveAndDelete(key string) ([]*net.UDPAddr, error) {
	sm.m.Lock()
	defer sm.m.Unlock()
	if sm.PendingRequests[key] == nil {
		return nil, errors.New("No entries matching key: " + key)
	}
	defer delete(sm.PendingRequests, key)
	return sm.PendingRequests[key], nil
}
