//go:build js && wasm
// +build js,wasm

package fetch_test

import (
	"context"
	"fmt"

	"github.com/monstercameron/GoWebComponents/fetch"
)

func ExampleUseFetch() {
	// Note: Hooks can only be used inside a component function

	// UseFetch is the low-level hook when you want raw fetch state.
	resource := fetch.UseFetch("https://api.example.com/data")

	state := resource.Get()

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
	resource.Refetch()
}

func ExampleUseResource() {
	// Note: Hooks can only be used inside a component function

	// UseResource is the preferred hook for typed, non-trivial loading.
	resource := fetch.UseResource(func(ctx context.Context) (int, error) {
		_ = ctx
		return 42, nil
	})

	state := resource.Get()
	if state.Loading {
		fmt.Println("Loading typed resource...")
		return
	}
	if state.Error != nil {
		fmt.Printf("Error: %v\n", state.Error)
		return
	}
	if state.Ready {
		fmt.Printf("Value: %d\n", state.Value)
	}

	resource.Reload()
	resource.Cancel()
}
