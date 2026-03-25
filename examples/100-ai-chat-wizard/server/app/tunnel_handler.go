//go:build !js && !wasm

package app

import (
	"log/slog"
	"net/http"

	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"google.golang.org/grpc"
)

func newGRPCTunnelHandler(grpcSrv *grpc.Server, logger *slog.Logger) http.Handler {
	return grpctunnel.Wrap(
		grpcSrv,
		grpctunnel.WithOriginCheck(func(r *http.Request) bool {
			// Allow all origins in development. Restrict to your domain in production.
			return true
		}),
		grpctunnel.WithConnectHook(func(r *http.Request) {
			logger.Info("tunnel: client connected", slog.String("remote_addr", r.RemoteAddr))
		}),
		grpctunnel.WithDisconnectHook(func(r *http.Request) {
			logger.Info("tunnel: client disconnected", slog.String("remote_addr", r.RemoteAddr))
		}),
	)
}
