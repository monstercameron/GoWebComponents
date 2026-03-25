//go:build js && wasm

package app

import "sync"

var grpcReconnectRegistry struct {
	mu      sync.RWMutex
	nextID  uint64
	id      uint64
	handler func(string)
}

func parseRegisterGRPCReconnectHandler(parseHandler func(string)) func() {
	grpcReconnectRegistry.mu.Lock()
	grpcReconnectRegistry.nextID++
	parseRegisteredID := grpcReconnectRegistry.nextID
	grpcReconnectRegistry.id = parseRegisteredID
	grpcReconnectRegistry.handler = parseHandler
	grpcReconnectRegistry.mu.Unlock()
	return func() {
		grpcReconnectRegistry.mu.Lock()
		if grpcReconnectRegistry.id == parseRegisteredID && grpcReconnectRegistry.handler != nil {
			grpcReconnectRegistry.id = 0
			grpcReconnectRegistry.handler = nil
		}
		grpcReconnectRegistry.mu.Unlock()
	}
}

func parseRequestGRPCReconnect(parseReason string) {
	grpcReconnectRegistry.mu.RLock()
	parseHandler := grpcReconnectRegistry.handler
	grpcReconnectRegistry.mu.RUnlock()
	if parseHandler != nil {
		parseHandler(parseReason)
	}
}
