package main

import (
	"fmt"
	html "goHTML/components"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	
    fmt.Println("WASM Go Initialized")
    html.Example1()
    fmt.Println("Main function completed")

	wg.Wait()
}
