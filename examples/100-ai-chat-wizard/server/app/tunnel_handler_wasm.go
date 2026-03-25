//go:build js && wasm

package app

import (
	"log/slog"
	"net/http"

	"google.golang.org/grpc"
)

func parseNewGRPCTunnelHandler(_ *grpc.Server, parseLogger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(parseW http.ResponseWriter, _ *http.Request) {
		parseLogger.Warn("tunnel: websocket gRPC bridge is unavailable for js/wasm server builds")
		http.Error(parseW, "websocket gRPC tunnel unavailable for js/wasm", http.StatusNotImplemented)
	})
}
