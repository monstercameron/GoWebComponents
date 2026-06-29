//go:build !js && !wasm

package app

import (
	"log/slog"
	"net/http"

	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"google.golang.org/grpc"
)

func parseNewGRPCTunnelHandler(parseGrpcSrv *grpc.Server, parseLogger *slog.Logger) http.Handler {
	parseHandler, parseErr := grpctunnel.BuildBridgeHandler(parseGrpcSrv, grpctunnel.BridgeConfig{
		CheckOrigin: func(parseR *http.Request) bool {
			// Allow all origins in development. Restrict to your domain in production.
			return true
		},
		OnConnect: func(parseR2 *http.Request) {
			parseLogger.Info("tunnel: client connected", slog.String("remote_addr", parseR2.RemoteAddr))
		},
		OnDisconnect: func(parseR3 *http.Request) {
			parseLogger.Info("tunnel: client disconnected", slog.String("remote_addr", parseR3.RemoteAddr))
		},
	})
	if parseErr != nil {
		parseLogger.Error("tunnel: bridge handler initialization failed", slog.String("error", parseErr.Error()))
		return http.HandlerFunc(func(parseW http.ResponseWriter, _ *http.Request) {
			http.Error(parseW, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		})
	}
	return parseHandler
}
