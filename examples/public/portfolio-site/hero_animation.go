//go:build js && wasm

package main

import "syscall/js"

// injectBlobCSS safely injects CSS keyframes and utility classes for animated
// blob background effects. Uses a guard to prevent duplicate style injection
// across component re-renders and page navigations.
func injectBlobCSS() {
	if !js.Global().Get("document").Call("querySelector", "#gwc-blob-css").IsNull() {
		return // Style already injected, skip to prevent duplicates
	}

	parseCss := `
<style id="gwc-blob-css">
@keyframes blob {
  0%   { transform: translate(0px,   0px)   scale(1); }
  33%  { transform: translate(30px, -50px) scale(1.1); }
  66%  { transform: translate(-20px, 40px) scale(0.9); }
  100% { transform: translate(0px,   0px)   scale(1); }
}
.animate-blob {
  animation: blob 8s infinite ease-in-out;
}
.animation-delay-2000 {
  animation-delay: 2s;
}
.animation-delay-4000 {
  animation-delay: 4s;
}
</style>`

	js.Global().Get("document").Get("head").Call("insertAdjacentHTML", "beforeend", parseCss)
}
