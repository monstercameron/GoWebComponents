module agenthub

go 1.26.0

require github.com/monstercameron/GoWebComponents/v4 v4.0.0

replace github.com/monstercameron/GoWebComponents/v4 => ../..

require github.com/gorilla/websocket v1.5.3

require (
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)
