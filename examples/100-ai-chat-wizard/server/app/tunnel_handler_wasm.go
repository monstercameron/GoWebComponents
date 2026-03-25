//go:build js && wasm

package app

import (
	"log/slog"
	"net/http"

	"google.golang.org/grpc"
)

func newGRPCTunnelHandler(_ *grpc.Server, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		logger.Warn("tunnel: websocket gRPC bridge is unavailable for js/wasm server builds")
		http.Error(w, "websocket gRPC tunnel unavailable for js/wasm", http.StatusNotImplemented)
	})
}
