package html

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"syscall/js"
	"time"
)

var globalRand = rand.New(rand.NewSource(time.Now().UnixNano()))
var stateCounter uint64 = 0

// GenerateUUID generates a basic UUID-like string using the current time and random numbers
func GenerateUUID() string {
	timestamp := time.Now().UnixNano()
	randPart1 := globalRand.Int63()
	randPart2 := globalRand.Int63()
	return fmt.Sprintf("%x-%x-%x", timestamp, randPart1, randPart2)
}

// useState is a thread-safe structure to manage state values.
type useState[T any] struct {
	value T
	mutex sync.RWMutex
	tag   string
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

// UseState creates a new state with an initial value and returns a pointer to the state value
// and a setter function to update the state.
func UseState[T any](initialValue T) (*T, func(T), string) {
	uniqueCounter := atomic.AddUint64(&stateCounter, 1)
	uniqueID := GenerateUUID()
	tag := fmt.Sprintf("state-%d-%s", uniqueCounter, uniqueID)

	state := &useState[T]{
		value: initialValue,
		tag:   tag,
	}

	setter := func(newValue T) {
		fmt.Println("Setting state", tag, "to", newValue)
		state.set(newValue)
		updateElementsWithState(tag, newValue)
	}

	return &state.value, setter, tag
}

// updateElementsWithState updates all elements in the DOM that are associated with a specific state variable.
func updateElementsWithState(stateID string, newValue interface{}) {
	// Get all nodes that have the specified dependency from the virtual DOM
	dependentNodes := getNodesWithDependency(stateID)
	fmt.Printf("Found %d dependent nodes for state '%s'\n", len(dependentNodes), stateID)

	// Find all DOM elements that depend on the changed state
	selector := fmt.Sprintf("[data-dependencies~='%s']", stateID)
	elements := js.Global().Get("document").Call("querySelectorAll", selector)
	length := elements.Get("length").Int()
	fmt.Printf("Found %d DOM elements depending on state '%s'\n", length, stateID)

	// Create a map to track which nodes have been updated
	updatedNodes := make(map[string]bool)

	// First, update all DOM elements found
	for i := 0; i < length; i++ {
		element := elements.Index(i)
		nodeID := element.Get("dataset").Get("nodeId").String()

		node := findNodeByID(dependentNodes, nodeID)
		if node == nil {
			fmt.Printf("No matching virtual node found for DOM element with ID '%s'\n", nodeID)
			continue
		}

		updateNodeAndElement(node, element, nodeID)
		updatedNodes[nodeID] = true
	}

	// Then, check for any remaining nodes in the virtual DOM that weren't updated
	for _, node := range dependentNodes {
		nodeID := node.GetID()
		if !updatedNodes[nodeID] {
			// This node exists in our virtual DOM but not in the actual DOM
			// We need to render it and insert it into the DOM
			fmt.Printf("Node '%s' exists in virtual DOM but not in actual DOM. Rendering and inserting.\n", nodeID)
			nodeHTML, err := node.Render()
			if err != nil {
				fmt.Printf("Error rendering node with ID '%s': %v\n", nodeID, err)
				continue
			}

			// Create a new element and insert it into the DOM
			// The exact insertion logic will depend on your DOM structure
			document := js.Global().Get("document")
			tempDiv := document.Call("createElement", "div")
			tempDiv.Set("innerHTML", nodeHTML)
			newElement := tempDiv.Get("firstChild")
			document.Get("body").Call("appendChild", newElement)

			fmt.Printf("New element inserted for node ID: %s\n", nodeID)
		}
	}
}

func findNodeByID(nodes []Node, id string) Node {
	for _, node := range nodes {
		if node.GetID() == id {
			return node
		}
	}
	return nil
}

func updateNodeAndElement(node Node, element js.Value, nodeID string) {
	// Render the updated HTML for the node
	nodeHTML, err := node.Render()
	if err != nil {
		fmt.Printf("Error rendering node with ID '%s': %v\n", nodeID, err)
		return
	}

	// Update the innerHTML of the DOM element
	element.Set("innerHTML", nodeHTML)
	fmt.Printf("Element updated for node ID: %s\n", nodeID)
}