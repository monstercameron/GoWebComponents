module livereload

go 1.26.0

require github.com/monstercameron/GoWebComponents v0.0.0

replace github.com/monstercameron/GoWebComponents => ../..

require (
	github.com/fsnotify/fsnotify v1.7.0
	github.com/gorilla/websocket v1.5.3
)

require golang.org/x/sys v0.45.0 // indirect
