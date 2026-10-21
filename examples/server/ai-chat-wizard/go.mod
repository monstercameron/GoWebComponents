module github.com/monstercameron/GoWebComponents/v7/examples/server/ai-chat-wizard

go 1.27.0

toolchain go1.27.1

require (
	github.com/alecthomas/chroma/v2 v2.27.0
	github.com/andybalholm/brotli v1.2.6
	github.com/anthropics/anthropic-sdk-go v1.27.1
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/joho/godotenv v1.5.1
	github.com/monstercameron/GoGRPCBridge v0.0.0-00010101000000-000000000000
	github.com/monstercameron/GoWebComponents/v7 v7.0.0
	github.com/monstercameron/GoWebComponents/v7/tools/agenthub v0.0.0
	github.com/ncruces/go-sqlite3 v0.32.0
	github.com/openai/openai-go v1.12.0
	github.com/yuin/goldmark v1.7.17
	github.com/yuin/goldmark-highlighting/v2 v2.0.0-20230729083705-37449abec8cc
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/trace v1.43.0
	golang.org/x/crypto v0.57.0
	golang.org/x/net v0.60.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478
	google.golang.org/grpc v1.82.1
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dlclark/regexp2/v2 v2.2.1 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/tetratelabs/wazero v1.11.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)

replace github.com/monstercameron/GoGRPCBridge => ../../../third_party/GoGRPCBridge

replace github.com/monstercameron/GoWebComponents/v7 => ../../..

replace github.com/monstercameron/GoWebComponents/v7/tools/agenthub => ../../../tools/agenthub
