package runtime

import (
	"errors"
	"fmt"
	"strings"
)

// ErrAgentRefStale reports that an agent node ref no longer resolves in the
// current fiber tree (the component unmounted or the tree was replaced). The
// agent bridge maps it to the stale-ref wire error so callers re-query
// instead of retrying the same ref.
var ErrAgentRefStale = errors.New("agent ref does not resolve in the current tree")

// ErrAgentRefInvalid reports a malformed agent ref (for example an empty
// string). The agent bridge maps it to the bad-payload wire error.
var ErrAgentRefInvalid = errors.New("agent ref is invalid")

// AgentRefForFiber returns the stable agent ref for a live fiber: the same
// key/index-disambiguated path the hot reload snapshot keys component state
// by, so a ref survives re-renders that recreate the fiber. The root fiber
// has no ref (empty string) - agents address nodes, not the mount point.
func AgentRefForFiber(parseFiber *Fiber) string {
	return hotReloadFiberPath(parseFiber)
}

// agentRefFrame pairs a fiber with its accumulated agent ref during tree walks.
type agentRefFrame struct {
	fiber *Fiber
	path  string
}

// ResolveAgentRef resolves a stable agent ref to its live fiber under the
// scheduler lock (the same discipline as Inspect). Resolution is read-only:
// it never marks fibers dirty or touches hook state.
func (parseRt *Runtime) ResolveAgentRef(parseRef string) (*Fiber, error) {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	return parseRt.resolveAgentRefLocked(parseRef)
}

// resolveAgentRefLocked resolves an agent ref assuming schedulerMu is already
// held, so bridge command executors can resolve inside their own locked
// sections without re-entering the mutex.
func (parseRt *Runtime) resolveAgentRefLocked(parseRef string) (*Fiber, error) {
	parseTrimmed := strings.TrimSpace(parseRef)
	if parseTrimmed == "" {
		return nil, fmt.Errorf("empty ref: %w", ErrAgentRefInvalid)
	}
	if parseRt == nil || parseRt.currentRoot == nil {
		return nil, fmt.Errorf("ref %q: no mounted tree: %w", parseTrimmed, ErrAgentRefStale)
	}

	// Walk with an explicit stack (deep trees, no recursion), descending only
	// into children whose accumulated path is a prefix of the wanted ref.
	parseStack := []agentRefFrame{{fiber: parseRt.currentRoot, path: ""}}
	for len(parseStack) > 0 {
		parseCurrent := parseStack[len(parseStack)-1]
		parseStack = parseStack[:len(parseStack)-1]
		if parseCurrent.path == parseTrimmed {
			return parseCurrent.fiber, nil
		}
		for parseChild := parseCurrent.fiber.child; parseChild != nil; parseChild = parseChild.sibling {
			parseSegment := hotReloadFiberPathSegment(parseChild)
			if parseSegment == "" {
				continue
			}
			parseChildPath := joinAgentRefPath(parseCurrent.path, parseSegment)
			if parseChildPath == parseTrimmed || strings.HasPrefix(parseTrimmed, parseChildPath+"/") {
				parseStack = append(parseStack, agentRefFrame{fiber: parseChild, path: parseChildPath})
			}
		}
	}
	return nil, fmt.Errorf("ref %q: %w", parseTrimmed, ErrAgentRefStale)
}

// joinAgentRefPath appends one path segment to an accumulated agent ref,
// matching hotReloadFiberPath's separator exactly.
func joinAgentRefPath(parseParent string, parseSegment string) string {
	if parseParent == "" {
		return parseSegment
	}
	return parseParent + "/" + parseSegment
}
