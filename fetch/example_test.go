//go:build js && wasm
// +build js,wasm

package fetch_test

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/fetch"
)

func ExampleUseFetch() {
	// Note: Hooks can only be used inside a component function

	// Fetch data from an API
	getState, refetch := fetch.UseFetch("https://api.example.com/data")

	state := getState()

	if state.Loading {
		fmt.Println("Loading...")
		return
	}

	if state.Error != "" {
		fmt.Printf("Error: %s\n", state.Error)
		return
	}

	// Use the data
	fmt.Printf("Data: %v\n", state.Data)

	// Manually trigger a refetch
	refetch()
}
