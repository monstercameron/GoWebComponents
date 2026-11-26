module github.com/monstercameron/GoWebComponents/v8/tools/livereload

go 1.27.0

replace github.com/monstercameron/GoWebComponents/v8/tools/agenthub => ../agenthub

replace github.com/monstercameron/GoWebComponents/v8/tools => ..

replace github.com/monstercameron/GoWebComponents/v8 => ../..

require (
	github.com/fsnotify/fsnotify v1.10.1
	github.com/gorilla/websocket v1.5.4-0.20240701034025-d67f41855da4
	github.com/monstercameron/GoWebComponents/v8 v8.0.0
	github.com/monstercameron/GoWebComponents/v8/tools v0.0.0
	github.com/monstercameron/GoWebComponents/v8/tools/agenthub v0.0.0
)

require (
	github.com/fxamacker/cbor/v2 v2.10.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/net v0.63.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
)
