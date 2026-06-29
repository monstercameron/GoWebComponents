//go:build js && wasm

package css_test

import (
	"strings"
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/css"
)

// fakeDOMSource installs a minimal document on globalThis so the wasm DOM sink
// has something to inject into under the node-based wasm test host (which has no
// real DOM). It supports the small surface sink_wasm.go touches: createElement,
// getElementById, an id setter that registers elements, head.appendChild,
// setAttribute/getAttribute, textContent, and querySelectorAll for the
// data-gwc-css blocks SeedFromDocument reads.
const fakeDOMSource = `
(function(){
  var store = {};
  function makeEl(tag){
    var el = { tagName: tag, textContent: "", _attrs: {} };
    el.setAttribute = function(k,v){ this._attrs[k] = String(v); };
    el.getAttribute = function(k){ return Object.prototype.hasOwnProperty.call(this._attrs,k) ? this._attrs[k] : null; };
    el.appendChild = function(c){ return c; };
    Object.defineProperty(el, "id", {
      get: function(){ return this._id; },
      set: function(v){ this._id = v; store[v] = this; },
      configurable: true
    });
    return el;
  }
  globalThis.__gwcMakeEl = makeEl;
  globalThis.document = {
    head: makeEl("head"),
    createElement: function(t){ return makeEl(t); },
    getElementById: function(id){ return Object.prototype.hasOwnProperty.call(store,id) ? store[id] : null; },
    querySelectorAll: function(sel){
      var matches = [];
      for (var k in store){
        var e = store[k];
        if (e._attrs && Object.prototype.hasOwnProperty.call(e._attrs, "data-gwc-css")) matches.push(e);
      }
      return { length: matches.length, item: function(i){ return matches[i]; } };
    }
  };
})();
`

func installFakeDOM(t *testing.T) {
	t.Helper()
	js.Global().Call("eval", fakeDOMSource)
	if !js.Global().Get("document").Truthy() {
		t.Fatal("fake document was not installed")
	}
}

func TestWasmSinkInjectsIntoStyleElement(t *testing.T) {
	installFakeDOM(t)
	css.Reset()

	class := css.New(css.Display.Flex, css.Gap(css.Px(8)))
	got := css.Harvest()
	if !strings.Contains(got, "."+string(class)+"{display:flex;gap:8px;}") {
		t.Fatalf("wasm sink did not inject the rule; harvested: %q", got)
	}

	// Dedup: re-injecting the same rule must not grow the style element.
	before := css.Harvest()
	for i := 0; i < 5; i++ {
		css.New(css.Display.Flex, css.Gap(css.Px(8)))
	}
	if after := css.Harvest(); after != before {
		t.Fatalf("dedup failed: style element grew\nbefore: %q\nafter:  %q", before, after)
	}
}

func TestWasmSeedFromDocumentSuppressesReinjection(t *testing.T) {
	installFakeDOM(t)
	css.Reset()

	// Compute the class for a grid rule without touching the DOM sink.
	prev := css.SetSink(sinkFunc(func(string, string) {}))
	gridClass := css.New(css.Display.Grid)
	css.SetSink(prev)

	css.Reset()

	// Simulate a server-rendered <style data-gwc-css="<gridClass>"> block.
	block := js.Global().Call("__gwcMakeEl", "style")
	block.Set("id", "server-block")
	block.Call("setAttribute", "data-gwc-css", string(gridClass))

	css.SeedFromDocument()

	// The grid rule is now seeded, so New must not inject it.
	again := css.New(css.Display.Grid)
	if again != gridClass {
		t.Fatalf("seeded class changed: %q != %q", again, gridClass)
	}
	if strings.Contains(css.Harvest(), "display:grid") {
		t.Fatalf("seeded rule was re-injected: %q", css.Harvest())
	}
}

type sinkFunc func(class, cssText string)

func (f sinkFunc) Emit(class, cssText string) { f(class, cssText) }
