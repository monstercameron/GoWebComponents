module livereload

go 1.26.0

require (
	agenthub v0.0.0
	github.com/monstercameron/GoWebComponents/v5 v5.0.0
)

replace agenthub => ../agenthub

replace github.com/monstercameron/GoWebComponents/v5 => ../..

require (
	github.com/fsnotify/fsnotify v1.7.0
	github.com/gorilla/websocket v1.5.3
)

require (
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/sys v0.45.0 // indirect
)
