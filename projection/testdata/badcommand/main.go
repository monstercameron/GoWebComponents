// Package main is the NEGATIVE half of P3.7 criterion (d)'s compile check.
//
// It misspells the command at the call site. This must NOT compile — that is
// the criterion. With a string-keyed API the same typo compiles cleanly and
// fails in the worker, after the user clicked something.
package main

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v4/projection"
)

type createItemArgs struct {
	Name string
}

type createItemResult struct {
	ID int64
}

var createItem = projection.Define[createItemArgs, createItemResult]("createItem")

func main() {
	var client projection.CommandClient
	var codec projection.Codec

	// "creatItem" — one letter missing. An undefined identifier, not a runtime
	// surprise.
	result, err := creatItem.Invoke(context.Background(), client, codec, createItemArgs{Name: "x"})
	_ = result
	_ = err
	_ = createItem
}
