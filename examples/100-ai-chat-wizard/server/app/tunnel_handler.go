//go:build !js && !wasm

package app

import (
	"log/slog"
	"net/http"

	"github.com/monstercameron/grpc-tunnel/pkg/grpctunnel"
	"google.golang.org/grpc"
)

func parseNewGRPCTunnelHandler(parseGrpcSrv *grpc.Server, parseLogger *slog.Logger) http.Handler {
	return grpctunnel.Wrap(
		parseGrpcSrv,
		grpctunnel.WithOriginCheck(func(parseR *http.Request) bool {
			// Allow all origins in development. Restrict to your domain in production.
			return true
		}),
		grpctunnel.WithConnectHook(func(parseR2 *http.Request) {
			parseLogger.Info("tunnel: client connected", slog.String("remote_addr", parseR2.RemoteAddr))
		}),
		grpctunnel.WithDisconnectHook(func(parseR3 *http.Request) {
			parseLogger.Info("tunnel: client disconnected", slog.String("remote_addr", parseR3.RemoteAddr))
		}),
	)
}
