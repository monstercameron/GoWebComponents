module example.com/gwc-wails-counter

go 1.27.0

require (
	github.com/monstercameron/GoWebComponents/v8 v8.0.0
	github.com/monstercameron/GoWebComponents/v8/desktop/wails v0.0.0
	github.com/wailsapp/wails/v3 v3.0.0-beta.17
	golang.org/x/sys v0.50.0
)

require (
	github.com/adrg/xdg v0.5.3 // indirect
	github.com/coder/websocket v1.8.14 // indirect
	github.com/fxamacker/cbor/v2 v2.10.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/ncruces/go-sqlite3 v0.34.4 // indirect
	github.com/ncruces/go-sqlite3-wasm/v2 v2.6.35302 // indirect
	github.com/ncruces/julianday v1.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/net v0.63.0 // indirect
)

replace github.com/monstercameron/GoWebComponents/v8 => ../../..

replace github.com/monstercameron/GoWebComponents/v8/desktop/wails => ../../../desktop/wails

replace github.com/wailsapp/wails/v3 => ../../../third_party/wails/v3
