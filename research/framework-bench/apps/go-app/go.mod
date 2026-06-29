// Separate module so the parent GoWebComponents module's build never tries to
// compile this (its deps live outside this repo; run `go mod tidy` to resolve).
module billsplitter/goapp

go 1.26

require github.com/maxence-charriere/go-app/v10 v10.1.8
