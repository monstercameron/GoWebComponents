package main

import (
	"fmt"
	html "goHTML/components"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	fmt.Println("Starting the example 1")
	html.Example2()

	wg.Wait()
}
