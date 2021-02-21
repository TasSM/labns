package defs

type DNSService interface {
	StartListener()
	StartDispatcher()
}
