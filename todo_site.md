# Personal Website 2025 - Todo List

## Build Scripts
- [x] Create `scripts/personal_website_2025/` directory
- [x] Create `scripts/personal_website_2025/build-go-compiler.sh`
- [x] Create `scripts/personal_website_2025/build-go-compiler.ps1`
- [x] Create `scripts/personal_website_2025/build-runtime.sh`
- [x] Create `scripts/personal_website_2025/build-runtime.ps1`
- [x] Create `scripts/personal_website_2025/deploy.sh`
- [x] Create `scripts/personal_website_2025/clean.sh`

## HTML Page
- [ ] Create `static/personal_website_2025.html`
- [ ] Add loading spinner and progress bar
- [ ] Add stage indicators for 7-stage loading process
- [ ] Add error display with dev-friendly messages
- [ ] Add app container for compiled output

## Go Compiler Integration (HEAVY LIFT)
- [ ] Research Go compiler cross-compilation to WASM
- [ ] Build simplified Go compiler or pattern matcher
- [ ] Implement virtual filesystem interface for compiler
- [ ] Add compilation error handling and reporting
- [ ] Test with GoWebComponents patterns

## JavaScript Infrastructure
- [ ] Create `static/script/personal-website-2025-cache-manager.js`
- [ ] Create `static/script/personal-website-2025-compiler-loader.js`
- [ ] Create `static/script/personal-website-2025-github-fetcher.js`
- [ ] Create `static/script/personal-website-2025-virtual-fs.js`
- [ ] Create `static/script/personal-website-2025-compilation-manager.js`
- [ ] Create `static/script/personal-website-2025-app-runner.js`

## Cache Management Features
- [ ] Implement IndexedDB cache system
- [ ] Add GitHub commit hash-based cache busting
- [ ] Add source file caching
- [ ] Add compiled WASM caching

## GitHub Source Fetching
- [ ] Implement repo structure fetching from GitHub API
- [ ] Add fetching for `examples/blog_landing_page.go`
- [ ] Add fetching for entire `fiber/` directory
- [ ] Add fetching for `go.mod` and `go.sum`
- [ ] Add error handling for network failures

## Virtual Filesystem
- [ ] Create in-memory filesystem for Go compiler
- [ ] Map GitHub files to virtual file paths
- [ ] Implement file operations (read, write, exists, listDir)
- [ ] Create Go package structure discovery
- [ ] Add import path resolution

## WASM Execution
- [ ] Implement WASM loading from compilation output
- [ ] Add Go runtime initialization for compiled app
- [ ] Handle runtime errors and debugging
- [ ] Mount compiled app to DOM

## Error Handling & UX
- [ ] Add detailed error messages for each stage
- [ ] Implement retry mechanisms
- [ ] Add progress reporting for long operations
- [ ] Create developer-friendly debugging output 