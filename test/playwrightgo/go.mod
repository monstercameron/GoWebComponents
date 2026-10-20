module github.com/monstercameron/GoWebComponents/v7/test/playwrightgo

go 1.27.0

toolchain go1.27.1

require (
	github.com/monstercameron/GoGRPCBridge v0.0.0-00010101000000-000000000000
	github.com/monstercameron/GoWebComponents/v7 v7.0.0
	github.com/monstercameron/GoWebComponents/v7/examples/server/ai-chat-wizard v0.0.0
	github.com/mxschmitt/playwright-go v0.6201.1
	github.com/ncruces/go-sqlite3 v0.32.0
	golang.org/x/crypto v0.51.0
	google.golang.org/grpc v1.82.1
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/deckarep/golang-set/v2 v2.8.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/tetratelabs/wazero v1.11.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel v1.43.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/trace v1.43.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
)

replace github.com/monstercameron/GoWebComponents/v7 => ../..

replace github.com/monstercameron/GoWebComponents/v7/examples/server/ai-chat-wizard => ../../examples/server/ai-chat-wizard

replace github.com/monstercameron/GoGRPCBridge => ../../third_party/GoGRPCBridge
