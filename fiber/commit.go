//go:build js && wasm
// +build js,wasm

package fiber

import (
	"fmt"
	"syscall/js"
)

// commitRoot commits the changes to the DOM
func commitRoot() {
	// fmt.Println("commitRoot: Starting to commit changes to DOM")
	startSchedulerSection()
	defer endSchedulerSection()

	for _, deletion := range deletions {
		// fmt.Printf("commitRoot: Processing deletion for fiber type %v\n", deletion.typeOf)
		commitWork(deletion)
	}
	if wipRoot.child != nil {
		// fmt.Printf("commitRoot: Committing child fiber of type %v\n", wipRoot.child.typeOf)
		commitWork(wipRoot.child)
	}
	currentRoot = wipRoot
	wipRoot = nil
	deletions = nil
	updateScheduled = false // Reset flag after commit
	// fmt.Println("commitRoot: Finished committing changes to DOM")

	// Execute effects after committing
	executeEffects()

	// Check memory pressure after each commit cycle
	checkMemoryPressure()
}

// executeEffects runs all effects collected during the render phase
func executeEffects() {
	if currentRoot == nil {
		return
	}

	var effectFibers []*Fiber
	var collectEffects func(fiber *Fiber)
	collectEffects = func(fiber *Fiber) {
		if fiber == nil {
			return
		}
		if len(fiber.effects) > 0 {
			effectFibers = append(effectFibers, fiber)
		}
		collectEffects(fiber.child)
		collectEffects(fiber.sibling)
	}

	// Collect fibers with effects starting from the root
	collectEffects(currentRoot.child)

	// Execute effects sequentially to avoid race conditions and ensure predictable order
	for _, fiber := range effectFibers {
		for _, effect := range fiber.effects {
			if effect != nil {
				effect()
			}
		}
		// Clear the effects after executing them to prevent accumulation
		fiber.effects = nil // Set to nil instead of empty slice to release memory
	}

	// Clear the slice to prevent memory leaks
	effectFibers = nil
}

// commitWork recursively commits work to the DOM
func commitWork(fiber *Fiber) {
	if fiber == nil {
		return
	}
	var domParentFiber = fiber.parent
	for domParentFiber != nil && (domParentFiber.dom.IsUndefined() || domParentFiber.dom.IsNull()) {
		domParentFiber = domParentFiber.parent
	}
	if domParentFiber == nil {
		// fmt.Println("commitWork: No valid parent DOM fiber found")
		return
	}
	domParent := domParentFiber.dom

	switch fiber.effectTag {
	case "PLACEMENT":
		if !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
			// fmt.Printf("commitWork: Appending child %v to parent %v\n", fiber.dom, domParent)
			domParent.Call("appendChild", fiber.dom)
		} else {
			// fmt.Println("commitWork: Fiber has no DOM node, committing its children")
			commitWork(fiber.child)
			return
		}
	case "UPDATE":
		if !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
			// fmt.Printf("commitWork: Updating DOM node for fiber type %v\n", fiber.typeOf)
			updateDom(fiber.dom, fiber.alternate.props, fiber.props)
		}
	case "DELETION":
		// fmt.Println("commitWork: Deleting DOM node")
		commitDeletion(fiber, domParent)
		return
	}

	// Commit children and siblings
	commitWork(fiber.child)
	commitWork(fiber.sibling)
}

// commitDeletion handles the deletion of DOM nodes and cleanup
func commitDeletion(fiber *Fiber, domParent js.Value) {
	// Enhanced cleanup for deleted components
	if fiber.hooks != nil {
		fmt.Printf("🧹 [COMPONENT_CLEANUP] Cleaning up hooks for deleted component\n")

		// Release event callbacks associated with this fiber
		for _, state := range fiber.hooks.state {
			if fn, ok := state.(js.Func); ok {
				fmt.Printf("🧹 [COMPONENT_CLEANUP] Releasing js.Func from state\n")
				fn.Release()
			}
		}

		// Release hooks back to pool
		releaseHooks(fiber.hooks)
		fiber.hooks = nil
	}

	// Clear effects to prevent memory leaks
	if len(fiber.effects) > 0 {
		fmt.Printf("🧹 [COMPONENT_CLEANUP] Clearing %d effects for deleted component\n", len(fiber.effects))
		fiber.effects = nil
	}

	if !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
		// fmt.Printf("commitDeletion: Removing child %v from parent %v\n", fiber.dom, domParent)
		domParent.Call("removeChild", fiber.dom)
	} else if fiber.child != nil {
		// fmt.Println("commitDeletion: Deleting child fibers recursively")
		commitDeletion(fiber.child, domParent)
	}

	// Recursively cleanup children and siblings
	if fiber.child != nil {
		commitDeletion(fiber.child, domParent)
	}
	if fiber.sibling != nil {
		commitDeletion(fiber.sibling, domParent)
	}
}
