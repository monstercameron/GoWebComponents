package runtime

import (
	"strings"
	"testing"
)

func TestSSRHookOwnerIsLazyUntilComponentRender(t *testing.T) {
	parseCtx := withSSRAtomScope(ssrHookOwnerContext(nil), ssrAtomScopeForRender(nil))
	if _, isLazy := parseCtx[ssrHookOwnerContextKey].(struct{}); !isLazy {
		t.Fatalf("top-level SSR context eagerly resolved owner: %#v", parseCtx[ssrHookOwnerContextKey])
	}

	parseRoot := &Element{Type: "div", Children: []any{
		&Element{Type: "TEXT_ELEMENT", TextContent: "ok"},
	}}
	var parseBuilder strings.Builder
	parseErr := renderElementToString(&parseBuilder, parseRoot, parseCtx)
	parseMarkup := parseBuilder.String()
	if parseErr != nil || parseMarkup != "<div>ok</div>" {
		t.Fatalf("host-only SSR render = %q, %v", parseMarkup, parseErr)
	}
	if _, isLazy := parseCtx[ssrHookOwnerContextKey].(struct{}); !isLazy {
		t.Fatalf("host-only render resolved owner: %#v", parseCtx[ssrHookOwnerContextKey])
	}

	withSSRHookFiber("component", nil, parseCtx, func() *Element {
		return &Element{Type: "TEXT_ELEMENT", TextContent: "component"}
	})
	// Native builds cache a non-zero goroutine owner here. JS/Wasm has no
	// goroutine identity, so getSSRHookOwner intentionally resolves to zero;
	// both platforms must still replace the lazy sentinel with a uint64 value.
	if _, isOwner := parseCtx[ssrHookOwnerContextKey].(uint64); !isOwner {
		t.Fatalf("component render did not resolve owner: %#v", parseCtx[ssrHookOwnerContextKey])
	}
}
