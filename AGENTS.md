AGENTS.md

Naming
Use verbSubject[Object].

Verbs
get set store cache clear render build handle filter format parse apply reset

Subjects
Use the owning domain:
sidebar composer thread model conv toolbar user canvas

Rules
- All new functions and variables start with a verb.
- Subject names the domain, not the data type.
- Add Object only when needed.
- Booleans start with is, has, can, or should.
- Every function needs a GoDoc comment.
- First word of each GoDoc comment must be the function name.
- Complex blocks need intent comments, not mechanics comments.

Examples
renderSidebar
renderSidebarConvRow
renderComposerInputArea
getModelLabel
storeConvList
cacheScrollPosition
handleUserDeleteConv
clearThreadMessages
buildToolbarSelectOption
isStreaming
hasExactCost

GWC
Run from repo root:
go run ./tools/gwc

Main commands
doctor bootstrap start examples dev serve build release test verify files import tailwind bench wasm dashboard seed

Primary docs
docs/GWC.md
docs/RUNNER_CONFIG.md
docs/TESTING.md
docs/PERFORMANCE.md
tools/README.md

Help
go run ./tools/gwc -h
go run ./tools/gwc <command> -h
go run ./tools/gwc wasm measure -h
go run ./tools/gwc wasm compare -h
go run ./tools/gwc bench -h

Flags (terse)
doctor: -json -host -port -audit(-policy/-baseline/-suppress/-write-baseline)
test: -lane -app/-main -root -json
build: -app/-main -root -profile -out/-output -json
dev: -app/-main -root -html/-index -wasm/-output -host -port -dry-run -json
serve: -root -host -port -index -wasm-file -wasm-route -fixture-json
release: -app/-main -root -out-dir -compression -post-link-opt -validate-smoke -json
verify: -app/-main -root -skip-tests -audit(-policy/-min-severity/-baseline/-suppress/-write-baseline) -json
examples/start/bootstrap: examples(-host,-port,-export-static-catalog) start(-mode,-init-git,-skip-*) bootstrap(-examples,-host,-port)
files/import/tailwind: files(-root,-ext,-exclude-dir,-json) import(-src,-out,-json) tailwind(-root,-input,-output,-manifest,-skip-manifest,-json)
bench/wasm/dashboard/seed: bench(-root,-lane,-bench,-count,-parallel,-out,-reference,-json) wasm(measure|compare|compare-compression|compare-cache|compare-toolchain; use subcommand -h) dashboard(-root,-status-url,-json) seed(-root,-command,-db-path,-json)

Typical usage
go run ./tools/gwc doctor
go run ./tools/gwc test -lane unit -lane wasm -lane hydration -lane browser
go run ./tools/gwc build -app .\examples\01-counter\main.go -root .\examples\01-counter
go run ./tools/gwc verify -app .\examples\01-counter\main.go -root .\examples\01-counter
go run ./tools/gwc examples
go run ./tools/gwc serve -root .\examples -port 8090
go run ./tools/gwc release -app .\examples\01-counter\main.go -root .\examples\01-counter

Todo execution
Do one todo at a time.

Per todo
1. Pick one unchecked item.
2. Read only needed files.
3. Make the smallest correct change.
4. Run the minimum validation.
5. Update todo status and notes.
6. Write a checkpoint.
7. Wait 10 seconds if supported, else continue.

Batching
Do not batch unrelated todos.
Combine only if:
- same subsection
- second is required for the first
- diff stays small
- same validation covers both

Loop
do one todo
validate
checkpoint
wait if supported
continue

Checkpoint
- completed todo
- files changed
- validation run
- result
- residual risk
- next suggested todo

Resume
Reopen the todo list, find the next unchecked item, continue from the last checkpoint, do not redo completed work.

Stop
Stop when no unchecked todos remain.

Priority
Controlled sequential progress beats throughput.

Bug fix workflow
1. Reproduce with the smallest reliable command, test, or browser flow.
2. Capture the exact failure.
3. Find root cause before editing.
4. Prefer the smallest root-cause fix.
5. Preserve public behavior unless the bug requires change.
6. Keep useful diagnostics.
7. Validate narrowly first, widen only if needed.

Implementation rules
- Do not guess when the repo can be inspected.
- Do not hide panics or errors just to pass tests.
- Do not remove useful debug detail without a good reason.
- Do not touch unrelated files or formatting.
- Respect nearby user changes.

Validation rules
- Prefer focused package tests, targeted Playwright specs, or the smallest reproducible command.
- For browser issues, capture the real console or page error.
- For wasm or cross-compilation, clear stale env vars first.
- Confirm the bug is fixed and diagnostics are clearer.
- Ad hoc binaries from repo root go under ./bin.

Examples
go test -c -o ./bin/<name>.test <package>
go build -o ./bin/<name>.exe <package>

Final report
- root cause
- what changed
- validation run
- remaining risk or follow-up
