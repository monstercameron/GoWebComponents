package html

import (
	"fmt"
	"math/rand"
	"sync"
	"syscall/js"
	"time"
)

// init initializes the random number generator with the current time
func init() {
	rand.Seed(time.Now().UnixNano())
}

// GenerateUUID generates a basic UUID-like string using the current time and random numbers
func GenerateUUID() string {
	// Get the current Unix timestamp in nanoseconds
	timestamp := time.Now().UnixNano()

	// Generate random numbers
	randPart1 := rand.Int63() // 63-bit random integer
	randPart2 := rand.Int63() // another 63-bit random integer

	// Combine the timestamp and random numbers to form a UUID-like string
	uuid := fmt.Sprintf("%x-%x-%x", timestamp, randPart1, randPart2)
	return uuid
}

// useState is a thread-safe structure to manage state values.
type useState[T any] struct {
	value T
	mutex sync.RWMutex
}

// get returns the current state value in a thread-safe manner.
func (s *useState[T]) get() T {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.value
}

// set updates the state value in a thread-safe manner.
func (s *useState[T]) set(newValue T) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.value = newValue
}

// UpdateStateElements updates all elements in the DOM that are associated with a specific state variable.
func updateElementsWithState(stateID string) {
	selector := fmt.Sprintf("[data-state~='%s']", stateID)
	elements := js.Global().Get("document").Call("querySelectorAll", selector)
	length := elements.Length()

	for i := 0; i < length; i++ {
		element := elements.Index(i)

		// Re-render the entire HTML content of the element associated with the stateID.
		nodeHTML, err := element.Get("stateNode").Call("Render")
		if err != nil {
			fmt.Println("Error rendering node:", err)
			continue
		}

		element.Set("innerHTML", nodeHTML)
	}
}


// UseState creates a new state with an initial value and returns a pointer to the state value
// and a setter function to update the state.
func UseState[T any](initialValue T) (*T, func(T), string) {
	state := &useState[T]{value: initialValue}
	uniqueID := GenerateUUID()               // Generate a unique UUID
	stateID := fmt.Sprintf("state-%p-%s", state, uniqueID) // Append UUID to stateID

	setter := func(newValue T) {
		state.set(newValue)
		updateElementsWithState(stateID)
	}

	return &state.value, setter, stateID
}

