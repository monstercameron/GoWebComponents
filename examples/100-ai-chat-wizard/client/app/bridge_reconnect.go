//go:build js && wasm

package app

import "sync"

var grpcReconnectRegistry struct {
	mu      sync.RWMutex
	nextID  uint64
	id      uint64
	handler func(string)
}

func registerGRPCReconnectHandler(handler func(string)) func() {
	grpcReconnectRegistry.mu.Lock()
	grpcReconnectRegistry.nextID++
	registeredID := grpcReconnectRegistry.nextID
	grpcReconnectRegistry.id = registeredID
	grpcReconnectRegistry.handler = handler
	grpcReconnectRegistry.mu.Unlock()
	return func() {
		grpcReconnectRegistry.mu.Lock()
		if grpcReconnectRegistry.id == registeredID && grpcReconnectRegistry.handler != nil {
			grpcReconnectRegistry.id = 0
			grpcReconnectRegistry.handler = nil
		}
		grpcReconnectRegistry.mu.Unlock()
	}
}

func requestGRPCReconnect(reason string) {
	grpcReconnectRegistry.mu.RLock()
	handler := grpcReconnectRegistry.handler
	grpcReconnectRegistry.mu.RUnlock()
	if handler != nil {
		handler(reason)
	}
}
