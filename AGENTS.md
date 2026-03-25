## Naming convention for functions and variables

All new function and variable names in this codebase follow the pattern:

```
verbSubject[Object]
```

- **verb** — what you do to/with it. Common: `get`, `set`, `store`, `cache`, `clear`, `render`, `build`, `handle`, `filter`, `format`, `parse`, `apply`, `reset`.
- **subject** — the owning domain or mechanism: `sidebar`, `composer`, `thread`, `model`, `conv`, `toolbar`, `user`, `canvas`.
- **object** *(optional)* — what within the subject the verb acts on, when not the subject itself: `ConvRow`, `InputArea`, `SelectOption`, `ScrollPosition`, `Messages`.

### Rules

1. Always start with a verb. `convList` or `sidebarPanel` are not allowed; use `getConvList` or `renderSidebarPanel`.
2. The subject is the domain that owns the thing, not the data type.
3. Add object only when the verb acts on something more specific than the subject.
4. Booleans start with `is`, `has`, `can`, or `should` (`isStreaming`, `hasExactCost`, `canSend`).
5. Every exported **and** unexported function must have a GoDoc comment. First word must be the function name.
6. Complex code blocks must have an inline comment stating **intent**, not mechanics — what the code is meant to achieve, not what it literally does. Prefer "// ensure the scroll anchor stays pinned after a new message lands" over "// call ScrollToBottom".

### Examples

| Name | Pattern | Reads as |
|---|---|---|
| `renderSidebar` | verb+subject | render the sidebar |
| `renderSidebarConvRow` | verb+subject+object | render the conv row in the sidebar |
| `renderComposerInputArea` | verb+subject+object | render the input area in the composer |
| `getModelLabel` | verb+subject+object | get the label for the model |
| `storeConvList` | verb+subject+object | store the conv list |
| `cacheScrollPosition` | verb+subject+object | cache the scroll position |
| `handleUserDeleteConv` | verb+subject+object | handle the user deleting a conv |
| `clearThreadMessages` | verb+subject+object | clear messages inside the thread |
| `buildToolbarSelectOption` | verb+subject+object | build the select option in the toolbar |
| `isStreaming` | bool prefix | whether streaming is active |
| `hasExactCost` | bool prefix | whether an exact cost is known |

---

## Execution pacing for long-running todo work

The agent must pace execution conservatively and avoid continuous high-intensity runs.

### Core rule
Work exactly one todo item at a time.

For each todo item, follow this sequence:

1. select one unchecked todo
2. inspect only the files needed for that todo
3. implement the smallest correct change
4. run the minimum required validation for that todo
5. update the todo file with status and notes
6. emit a checkpoint summary
7. wait 5 minutes before starting the next todo

### Required wait
After completing each todo item, the agent must wait **10 seconds** before beginning the next todo.

If the harness does not support waiting, the agent should continue immediately with the next unchecked todo instead of stopping.

### Harness directive
If the harness supports autonomous execution with delay or sleep semantics, use them.

Preferred harness behavior:
- complete one todo
- checkpoint
- sleep 10 seconds
- continue with the next unchecked todo
- repeat until all todos are complete

Equivalent pseudocode:

while todos_remaining:
    do_one_todo()
    write_checkpoint()
    sleep(10)

### Todo batching restriction
Do not complete multiple unrelated todos in the same execution burst.

Only combine todos when all of the following are true:
- they are in the same subsection
- the second is mechanically required to finish the first
- the diff remains small and reviewable
- the same validation covers both safely

Default behavior is always:
- one todo
- validate
- checkpoint
- wait 5 minutes
- continue

### Checkpoint format
After each todo, record:

- completed todo
- files changed
- validation run
- result
- residual risk
- next suggested todo

### Resume behavior
After the 5-minute wait, the agent should:
- re-open the todo list
- find the next unchecked todo
- continue from the last checkpoint
- avoid redoing already completed work

### Stop conditions
The agent must stop instead of continuing when:
- there are no more unchecked todos

### Priority
These pacing instructions override any generic preference for maximizing throughput.
For this project, controlled sequential progress is preferred over rapid continuous execution.

## Generic issue-fixing instructions

When the task is to fix a bug, regression, runtime panic, broken example, flaky test, or unclear diagnostic, the agent should use this default workflow unless the user asks for something narrower.

### Debugging workflow

1. reproduce the issue first using the smallest reliable command, test, or browser flow
2. capture the exact failing message, stack frame, or observable incorrect behavior
3. identify the root cause before editing files
4. prefer the smallest fix that addresses the root cause instead of adding a broad workaround
5. preserve existing public behavior unless the bug itself requires a behavior change
6. keep existing logs, panic text, or diagnostics when useful, but restructure them if readability is part of the fix
7. validate with the narrowest relevant test or command first, then widen only if needed

### Implementation rules

- do not guess about the failing path when the repo can be inspected directly
- do not silently swallow panics or errors just to make a test pass
- do not remove useful debugging detail when improving message design; keep the original failure signal visible when practical
- do not change unrelated files or reformat unrelated code while fixing the issue
- if there are user changes in nearby files, read them carefully and work with them instead of overwriting them

### Validation rules

- prefer targeted package tests, focused Playwright specs, or the smallest reproducible command
- if the failure involves browser output, capture the actual browser console or page error instead of paraphrasing it
- if the failure involves wasm or cross-compilation, clear stale environment variables before concluding the result
- after the fix, confirm both that the issue is gone and that the improved output is easier to interpret
- when compiling ad hoc binaries from the repo root, always write outputs under `./bin`:
  - use `go test -c -o ./bin/<name>.test <package>` instead of `go test -c <package>`
  - use `go build -o ./bin/<name>.exe <package>` (or `./bin/<name>` on non-Windows) instead of `go build <package>`

### Expected final report

When the fix is complete, report:

- what the root cause was
- what changed
- what validation was run
- any remaining risk or follow-up that would materially improve the area
