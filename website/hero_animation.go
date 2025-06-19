//go:build js && wasm
// +build js,wasm

package website

import "syscall/js"

// injectBlobCSS ensures the keyframes and utility classes for the "blob" background
// animation are present. It is safe to call multiple times; the style tag is only
// added once per session.
func injectBlobCSS() {
	if !js.Global().Get("document").Call("querySelector", "#gwc-blob-css").IsNull() {
		return // already injected
	}

	css := `
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

	js.Global().Get("document").Get("head").Call("insertAdjacentHTML", "beforeend", css)
}
