# Example 13: Go Compiler in Browser

This example demonstrates how to run the Go compiler (`cmd/compile`) and linker (`cmd/link`) directly in the browser using WebAssembly.

## How it works

1.  **WASM Binaries**: We compile the Go toolchain (`cmd/compile` and `cmd/link`) to WASM.
2.  **Virtual Filesystem**: We implement a virtual filesystem (VFS) in JavaScript to store source files and build artifacts.
3.  **FS Polyfill**: We patch the Node.js `fs` API (used by `wasm_exec.js`) to redirect file operations to our VFS.
4.  **Compilation Pipeline**:
    *   Fetch Go source code (e.g. from GitHub).
    *   Write source to VFS.
    *   Run `compile.wasm` to generate object files (`.o`).
    *   Run `link.wasm` to generate the final executable (`.wasm`).
    *   Instantiate and run the generated WASM.

## Setup

1.  Run `build-compiler.ps1` to build the compiler and linker WASM binaries.
2.  Serve the directory (e.g. using `go run ../../tools/serve.ps1`).
3.  Open `http://localhost:8080/examples/13-browser-compiler/`.

## Limitations

*   **Standard Library**: The compiler needs compiled package archives (`.a` files) for imported packages (like `fmt`, `os`). This example does not include the full Go standard library in the VFS, so compilation of programs importing standard packages will fail unless those archives are provided.
*   **Performance**: The compiler binaries are large (~20MB each) and take time to load and run.
