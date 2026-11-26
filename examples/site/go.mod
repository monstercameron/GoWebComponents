module github.com/monstercameron/GoWebComponents/v8/examples/site

go 1.27.0

toolchain go1.27.1

require (
	github.com/alecthomas/chroma/v2 v2.27.0
	github.com/monstercameron/GoWebComponents/v8 v8.0.0
)

require (
	github.com/dlclark/regexp2/v2 v2.2.1 // indirect
	github.com/fxamacker/cbor/v2 v2.10.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/net v0.63.0 // indirect
)

replace github.com/monstercameron/GoWebComponents/v8 => ../..
