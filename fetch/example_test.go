//go:build js && wasm

package fetch_test

import (
	"context"
	"fmt"

	"github.com/monstercameron/GoWebComponents/v6/fetch"
)

func ExampleUseFetch() {
	// Note: Hooks can only be used inside a component function

	// UseFetch is the low-level hook when you want raw fetch state.
	parseResource := fetch.UseFetch("https://api.example.com/data")

	parseState := parseResource.Get()

	if parseState.Loading {
		fmt.Println("Loading...")
		return
	}

	if parseState.Error != "" {
		fmt.Printf("Error: %s\n", parseState.Error)
		return
	}

	// Use the data
	fmt.Printf("Data: %v\n", parseState.Data)

	// Manually trigger a refetch
	parseResource.Refetch()
}

func ExampleUseResource() {
	// Note: Hooks can only be used inside a component function

	// UseResource is the preferred hook for typed, non-trivial loading.
	parseResource := fetch.UseResource(func(parseCtx context.Context) (int, error) {
		_ = parseCtx
		return 42, nil
	})

	parseState := parseResource.Get()
	if parseState.Loading {
		fmt.Println("Loading typed resource...")
		return
	}
	if parseState.Error != nil {
		fmt.Printf("Error: %v\n", parseState.Error)
		return
	}
	if parseState.Ready {
		fmt.Printf("Value: %d\n", parseState.Value)
	}

	parseResource.Reload()
	parseResource.Cancel()
}

func ExampleUseCachedResource() {
	// Note: Hooks can only be used inside a component function

	parseResource := fetch.UseCachedResource("users", func(parseCtx context.Context) ([]string, error) {
		_ = parseCtx
		return []string{"Ada", "Grace"}, nil
	}, fetch.CacheOptions{})

	parseState := parseResource.Get()
	if parseState.Loading && !parseState.Ready {
		fmt.Println("Loading shared cache...")
		return
	}
	if parseState.Error != nil {
		fmt.Printf("Error: %v\n", parseState.Error)
		return
	}
	if parseState.Ready {
		fmt.Printf("Cached users: %d\n", len(parseState.Value))
	}

	parseResource.Update(func(parsePrev []string) []string {
		return append(parsePrev, "Linus")
	})
	parseResource.Invalidate()
	parseResource.Reload()
}
