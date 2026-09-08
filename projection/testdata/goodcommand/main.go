// Package main is the POSITIVE half of P3.7 criterion (d)'s compile check.
//
// It exercises the command API correctly and must build. Without it, the
// negative test could pass because of an unrelated build error — a missing
// import, a renamed symbol — and would then be asserting nothing.
package main

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v6/projection"
)

type createItemArgs struct {
	Name string
}

type createItemResult struct {
	ID int64
}

// The name string appears exactly once in the program.
var createItem = projection.Define[createItemArgs, createItemResult]("createItem")

func main() {
	var client projection.CommandClient
	var codec projection.Codec

	// Correct spelling, correct argument type, correct result type.
	result, err := createItem.Invoke(context.Background(), client, codec, createItemArgs{Name: "x"})
	_ = result.ID
	_ = err
}
