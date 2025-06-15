//go:build js && wasm
// +build js,wasm

package fiber

import (
	"runtime"
	"syscall/js"
	"time"
)

// commitRoot commits the changes to the DOM
func commitRoot() {
	debugf("COMMIT", "🎯 commitRoot: starting commit phase\n")
	commitStartTime := time.Now()

	// Memory diagnostics before commit
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)
	debugf("COMMIT", "💾 commitRoot: memory before - heap: %d KB, objects: %d\n",
		memBefore.HeapAlloc/1024, memBefore.HeapObjects)

	// fmt.Println("commitRoot: Starting to commit changes to DOM")
	startSchedulerSection()
	defer endSchedulerSection()

	// Process deletions first
	debugf("COMMIT", "🗑️ commitRoot: processing %d deletions\n", len(deletions))
	deletionStartTime := time.Now()
	for i, deletion := range deletions {
		debugf("COMMIT", "🗑️ commitRoot: processing deletion %d/%d (type: %v)\n",
			i+1, len(deletions), deletion.typeOf)
		// fmt.Printf("commitRoot: Processing deletion for fiber type %v\n", deletion.typeOf)
		commitWork(deletion)
	}
	deletionDuration := time.Since(deletionStartTime)
	debugf("COMMIT", "✅ commitRoot: deletions completed in %v\n", deletionDuration)

	// Commit new/updated work
	if wipRoot.child != nil {
		debugf("COMMIT", "🔄 commitRoot: committing work tree starting from child %p (type: %v)\n",
			wipRoot.child, wipRoot.child.typeOf)
		workStartTime := time.Now()
		// fmt.Printf("commitRoot: Committing child fiber of type %v\n", wipRoot.child.typeOf)
		commitWork(wipRoot.child)
		workDuration := time.Since(workStartTime)
		debugf("COMMIT", "✅ commitRoot: work tree committed in %v\n", workDuration)
	} else {
		debugf("COMMIT", "📋 commitRoot: no child work to commit\n")
	}

	// Update root references
	debugf("COMMIT", "🔄 commitRoot: updating root references\n")
	currentRoot = wipRoot
	wipRoot = nil
	deletions = nil
	updateScheduled = false // Reset flag after commit
	// fmt.Println("commitRoot: Finished committing changes to DOM")

	// Execute effects after committing
	debugf("COMMIT", "⚡ commitRoot: executing effects\n")
	effectsStartTime := time.Now()
	executeEffects()
	effectsDuration := time.Since(effectsStartTime)
	debugf("COMMIT", "✅ commitRoot: effects executed in %v\n", effectsDuration)

	// Check memory pressure after each commit cycle
	debugf("COMMIT", "🔍 commitRoot: checking memory pressure\n")
	checkMemoryPressure()

	// Memory diagnostics after commit
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)
	debugf("COMMIT", "💾 commitRoot: memory after - heap: %d KB, objects: %d (diff: %+d KB, %+d objects)\n",
		memAfter.HeapAlloc/1024, memAfter.HeapObjects,
		int64(memAfter.HeapAlloc/1024)-int64(memBefore.HeapAlloc/1024),
		int64(memAfter.HeapObjects)-int64(memBefore.HeapObjects))

	commitDuration := time.Since(commitStartTime)
	debugf("COMMIT", "🎉 commitRoot: commit cycle completed in %v\n", commitDuration)
}

// executeEffects runs all effects collected during the render phase
func executeEffects() {
	debugf("COMMIT", "🎯 executeEffects: starting effect execution\n")

	if currentRoot == nil {
		debugf("COMMIT", "🚨 executeEffects: currentRoot is nil, aborting\n")
		return
	}

	// Use pooled slice to reduce allocations and GC pressure
	effectFibers := getEffectFibersSlice()
	defer returnEffectFibersSlice(effectFibers)
	
	effectCount := 0
	var collectEffects func(fiber *Fiber)
	collectEffects = func(fiber *Fiber) {
		if fiber == nil {
			return
		}
		if len(fiber.effects) > 0 {
			debugf("COMMIT", "📋 executeEffects: found %d effects in fiber %p (type: %v)\n",
				len(fiber.effects), fiber, fiber.typeOf)
			*effectFibers = append(*effectFibers, fiber)
			effectCount += len(fiber.effects)
		}
		collectEffects(fiber.child)
		collectEffects(fiber.sibling)
	}

	// Collect fibers with effects starting from the root
	debugf("COMMIT", "🔍 executeEffects: collecting effects from fiber tree\n")
	collectStartTime := time.Now()
	collectEffects(currentRoot.child)
	collectDuration := time.Since(collectStartTime)

	debugf("COMMIT", "📊 executeEffects: collected %d effects from %d fibers in %v\n",
		effectCount, len(*effectFibers), collectDuration)

	// Execute effects sequentially to avoid race conditions and ensure predictable order
	executionStartTime := time.Now()
	executedCount := 0
	for fiberIndex, fiber := range *effectFibers {
		debugf("COMMIT", "⚡ executeEffects: executing effects for fiber %d/%d (type: %v)\n",
			fiberIndex+1, len(*effectFibers), fiber.typeOf)

		for effectIndex, effect := range fiber.effects {
			if effect != nil {
				debugf("COMMIT", "🎬 executeEffects: running effect %d/%d\n",
					effectIndex+1, len(fiber.effects))
				effectStartTime := time.Now()
				effect()
				effectDuration := time.Since(effectStartTime)
				debugf("COMMIT", "✅ executeEffects: effect completed in %v\n", effectDuration)
				executedCount++
			} else {
				debugf("COMMIT", "🚨 executeEffects: found nil effect at index %d\n", effectIndex)
			}
		}
		// Clear the effects after executing them to prevent accumulation
		debugf("COMMIT", "🧹 executeEffects: clearing %d effects from fiber\n", len(fiber.effects))
		fiber.effects = nil // Set to nil instead of empty slice to release memory
	}

	executionDuration := time.Since(executionStartTime)
	debugf("COMMIT", "✅ executeEffects: executed %d effects in %v\n", executedCount, executionDuration)

	// Note: effectFibers slice will be automatically returned to pool via defer
}

// commitWork recursively commits work to the DOM
func commitWork(fiber *Fiber) {
	if fiber == nil {
		debugf("COMMIT", "🚨 commitWork: received nil fiber\n")
		return
	}

	debugf("COMMIT", "🔄 commitWork: processing fiber %p (type: %v, effectTag: %s)\n",
		fiber, fiber.typeOf, fiber.effectTag)

	var domParentFiber = fiber.parent
	parentSearchDepth := 0
	for domParentFiber != nil && (domParentFiber.dom.IsUndefined() || domParentFiber.dom.IsNull()) {
		domParentFiber = domParentFiber.parent
		parentSearchDepth++
		if parentSearchDepth > 100 { // Prevent infinite loops
			debugf("COMMIT", "🚨 commitWork: parent search depth exceeded, breaking\n")
			break
		}
	}

	if domParentFiber == nil {
		debugf("COMMIT", "🚨 commitWork: no valid parent DOM fiber found after searching %d levels\n", parentSearchDepth)
		// fmt.Println("commitWork: No valid parent DOM fiber found")
		return
	}

	domParent := domParentFiber.dom
	debugf("COMMIT", "🎯 commitWork: found DOM parent %v after searching %d levels\n",
		domParent.Type(), parentSearchDepth)

	switch fiber.effectTag {
	case "PLACEMENT":
		debugf("COMMIT", "➕ commitWork: PLACEMENT - placing new DOM node\n")
		if !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
			debugf("COMMIT", "📄 commitWork: appending DOM node %v to parent %v\n",
				fiber.dom.Type(), domParent.Type())
			// fmt.Printf("commitWork: Appending child %v to parent %v\n", fiber.dom, domParent)
			domParent.Call("appendChild", fiber.dom)
			debugf("COMMIT", "✅ commitWork: DOM node appended successfully\n")
		} else {
			debugf("COMMIT", "🚨 commitWork: fiber has no DOM node, committing children instead\n")
			// fmt.Println("commitWork: Fiber has no DOM node, committing its children")
			commitWork(fiber.child)
			return
		}

	case "UPDATE":
		debugf("COMMIT", "🔄 commitWork: UPDATE - updating existing DOM node\n")
		if !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
			debugf("COMMIT", "📝 commitWork: updating DOM properties\n")
			// fmt.Printf("commitWork: Updating DOM node for fiber type %v\n", fiber.typeOf)
			updateStartTime := time.Now()
			updateDom(fiber.dom, fiber.alternate.props, fiber.props)
			updateDuration := time.Since(updateStartTime)
			debugf("COMMIT", "✅ commitWork: DOM update completed in %v\n", updateDuration)
		} else {
			debugf("COMMIT", "🚨 commitWork: UPDATE fiber has no DOM node\n")
		}

	case "DELETION":
		debugf("COMMIT", "🗑️ commitWork: DELETION - removing DOM node\n")
		// fmt.Println("commitWork: Deleting DOM node")
		deletionStartTime := time.Now()
		commitDeletion(fiber, domParent)
		deletionDuration := time.Since(deletionStartTime)
		debugf("COMMIT", "✅ commitWork: deletion completed in %v\n", deletionDuration)
		return

	default:
		debugf("COMMIT", "❓ commitWork: unknown effect tag: %s\n", fiber.effectTag)
	}

	// Commit children and siblings
	if fiber.child != nil {
		debugf("COMMIT", "👶 commitWork: committing child %p\n", fiber.child)
		commitWork(fiber.child)
	}
	if fiber.sibling != nil {
		debugf("COMMIT", "👫 commitWork: committing sibling %p\n", fiber.sibling)
		commitWork(fiber.sibling)
	}

	debugf("COMMIT", "✅ commitWork: completed processing fiber %p\n", fiber)
}

// commitDeletion handles the deletion of DOM nodes and cleanup
func commitDeletion(fiber *Fiber, domParent js.Value) {
	debugf("COMMIT", "🗑️ commitDeletion: cleaning up fiber %p (type: %v)\n", fiber, fiber.typeOf)

	// Enhanced cleanup for deleted components
	if fiber.hooks != nil {
		debugf("COMMIT", "🧹 commitDeletion: cleaning up hooks for deleted component\n")

		// Release event callbacks associated with this fiber
		callbacksReleased := 0
		for i, state := range fiber.hooks.state {
			if fn, ok := state.(js.Func); ok {
				debugf("COMMIT", "🧹 commitDeletion: releasing js.Func from state[%d]\n", i)
				fn.Release()
				callbacksReleased++
			}
		}
		debugf("COMMIT", "📊 commitDeletion: released %d callbacks from hooks\n", callbacksReleased)

		// Release hooks back to pool
		debugf("COMMIT", "♻️ commitDeletion: returning hooks to pool\n")
		releaseHooks(fiber.hooks)
		fiber.hooks = nil
	}

	// Release event callbacks created by GoUseFunc
	if len(fiber.eventCallbacks) > 0 {
		debugf("COMMIT", "🧹 commitDeletion: releasing %d event callbacks from fiber\n", len(fiber.eventCallbacks))
		for i, callback := range fiber.eventCallbacks {
			debugf("COMMIT", "🧹 commitDeletion: releasing event callback %d\n", i)
			callback.Release()
		}
		fiber.eventCallbacks = nil
		debugf("COMMIT", "✅ commitDeletion: all event callbacks released\n")
	}

	// Clear effects to prevent memory leaks
	if len(fiber.effects) > 0 {
		debugf("COMMIT", "🧹 commitDeletion: clearing %d effects for deleted component\n", len(fiber.effects))
		fiber.effects = nil
	}

	if !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
		debugf("COMMIT", "📄 commitDeletion: attempting to remove DOM node %v from parent %v\n",
			fiber.dom.Type(), domParent.Type())
		
		// Check if the node is actually a child of the parent before removing
		// This prevents the "node to be removed is not a child" error
		children := domParent.Get("childNodes")
		isChild := false
		if !children.IsUndefined() && !children.IsNull() {
			length := children.Get("length").Int()
			for i := 0; i < length; i++ {
				child := children.Call("item", i)
				if child.Equal(fiber.dom) {
					isChild = true
					break
				}
			}
		}
		
		if isChild {
			debugf("COMMIT", "✅ commitDeletion: confirmed node is child, removing\n")
			domParent.Call("removeChild", fiber.dom)
			debugf("COMMIT", "✅ commitDeletion: DOM node removed successfully\n")
		} else {
			debugf("COMMIT", "⚠️ commitDeletion: node is not a direct child, skipping removal\n")
		}
	} else if fiber.child != nil {
		debugf("COMMIT", "🚨 commitDeletion: no DOM node, recursively deleting children\n")
		commitDeletion(fiber.child, domParent)
	}

	// Recursively cleanup children and siblings
	if fiber.child != nil {
		debugf("COMMIT", "👶 commitDeletion: recursively cleaning up child %p\n", fiber.child)
		commitDeletion(fiber.child, domParent)
	}
	if fiber.sibling != nil {
		debugf("COMMIT", "👫 commitDeletion: recursively cleaning up sibling %p\n", fiber.sibling)
		commitDeletion(fiber.sibling, domParent)
	}

	// --- ADDED: Reset and return fiber to pool for memory management ---
	resetFiber(fiber)
	fiberPool.Put(fiber)
	debugf("COMMIT", "♻️ commitDeletion: fiber reset and returned to pool %p\n", fiber)

	debugf("COMMIT", "✅ commitDeletion: completed cleanup for fiber %p\n", fiber)
}
