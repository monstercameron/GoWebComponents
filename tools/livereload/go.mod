module github.com/monstercameron/GoWebComponents/v7/tools/livereload

go 1.27.0

replace github.com/monstercameron/GoWebComponents/v7/tools/agenthub => ../agenthub

replace github.com/monstercameron/GoWebComponents/v7/tools => ..

replace github.com/monstercameron/GoWebComponents/v7 => ../..

require (
	github.com/fsnotify/fsnotify v1.10.1
	github.com/gorilla/websocket v1.5.3
	github.com/monstercameron/GoWebComponents/v7 v7.0.0
	github.com/monstercameron/GoWebComponents/v7/tools v0.0.0
	github.com/monstercameron/GoWebComponents/v7/tools/agenthub v0.0.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/sys v0.48.0 // indirect
)
