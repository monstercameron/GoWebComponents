package html

import (
	"fmt"
	"sync"
)

// Node interface represents any element in the HTML structure, including both text and element nodes.
type Node interface {
	Render() (string, error)
	GetDependencies() []string
	GetID() string
}

// NodeRegistry is a global registry to keep track of all nodes added to the HTML tree.
var nodeRegistry = struct {
	sync.RWMutex
	nodes map[string]Node
}{nodes: make(map[string]Node)}

// Node Registry Management Functions
// ----------------------------------

// registerNode adds a node to the global registry.
func registerNode(node Node) {
	nodeRegistry.Lock()
	defer nodeRegistry.Unlock()
	nodeRegistry.nodes[node.GetID()] = node
}

// unregisterNode removes a node from the global registry.
func unregisterNode(nodeID string) {
	nodeRegistry.Lock()
	defer nodeRegistry.Unlock()
	delete(nodeRegistry.nodes, nodeID)
}

// Dependency Management Functions
// -------------------------------

// getNodesWithDependency retrieves all nodes that depend on a specific state ID.
func getNodesWithDependency(stateID string) []Node {
	nodeRegistry.RLock()
	defer nodeRegistry.RUnlock()

	fmt.Println("State ID:", stateID)
	fmt.Println("Node Registry:", nodeRegistry.nodes)

	var dependentNodes []Node
	for _, node := range nodeRegistry.nodes {
			fmt.Println("\nChecking node:", node.GetID())
			fmt.Println("\t\tDependency:", node.GetDependencies())
		for _, dep := range node.GetDependencies() {
			fmt.Println("\t\t found Dependency:", dep)
			if dep == stateID {
				dependentNodes = append(dependentNodes, node)
				break
			}
		}
	}
	return dependentNodes
}

// getNodeByID retrieves a node from our virtual DOM by its ID
func getNodeByID(id string) Node {
	nodeRegistry.RLock()
	defer nodeRegistry.RUnlock()
	
	if node, exists := nodeRegistry.nodes[id]; exists {
		return node
	}
	return nil
}