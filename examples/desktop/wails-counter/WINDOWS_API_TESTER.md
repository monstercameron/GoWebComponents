# Windows API Lab

The example now opens a native Windows test console at `#/tester`. The original
counter, persistence and cancellation demo remains at `#/counter`. This is a
finite capability tester, not a claim to cover every Win32 API.

## Build and run

From the repository root:

```powershell
go run ./tools/gwc desktop build -root ./examples/desktop/wails-counter -json
.\examples\desktop\wails-counter\bin\wails-counter.exe
```

Use **Tests → Open second tester window** for caller-window ownership checks.
Close all windows before rebuilding. The tester uses the pinned Wails beta.17
source, not whichever globally installed Wails CLI happens to be available.

File-selection workflows now use the reusable SDK and isolated Wails adapter.
Run the rebuilt executable with `--file-dialogs=false` to disable all four lab
path selectors, the counter's legacy picker binding and the SDK binding. The
counter's picker control reflects capability availability; the lab deliberately
keeps its probes available so denied results can be inspected. **Report export
is a separate file-writing operation and is not disabled by this switch.** Menus,
clipboard and window APIs are also outside this specific switch. Library hosts
default to denied; this test app explicitly opts in by default.

## Test surfaces

| Area | Cases |
| --- | --- |
| Native context menu | Right-click target, command callback/context data, checkbox, exclusive radios, submenu, disabled command, Escape dismissal |
| Standard edit menu | Editable text, selection, native right-click editing menu, keyboard/IME input |
| Application menu/keys | Tests menu, Ctrl+Shift+K accelerator, F8 per-window callback, second tester window |
| Pickers | Single/multiple text files, Unicode filenames, directory, save-path selection, cancellation, caller-window ownership |
| Native messages | Information dismissal, question Yes/No, focus recovery |
| Windows | Size/info, resize, maximize/restore, fullscreen/exit; Escape exits fullscreen |
| Clipboard | Explicit synthetic-text write and fixture-match check; never automatic |
| Displays | Native screen enumeration |
| Evidence | Bounded session log, explicit visual verdicts, JSON export through native save dialog |

The private fixture directory contains `sample-a.txt`, `sample-b.txt`,
`sample-unicode-雪.txt` and `folder`. Picker selection returns paths only; it does
not read arbitrary selected file contents. **Save path** does not write a file.
**Export report** is the separate write action and must not overwrite an existing
file. Fixture files are removed when the host shuts down; export reports outside
the fixture directory if you want to retain them.

Clipboard writing replaces current clipboard contents and is opt-in. Reading
compares against the synthetic fixture without putting existing clipboard text
into the exported log. Reports may contain selected local paths and typed notes;
inspect them before sharing. Nothing is uploaded by the tester.

## Meaning of results

- `completed`: the native operation returned or a native callback ran. This is
  not a visual, accessibility or usability pass.
- `cancelled`: the operator cancelled the picker, or the call context
  was cancelled. It is not a successful selection.
- `error`: a real operation failed; diagnostics remain visible.
- `observed-pass` / `observed-fail`: an explicit operator verdict with a case ID
  and note. The tester does not manufacture these verdicts.
- `not-tested`: deliberately unverified; not a passing result.

The report belongs to this host session and is shared between its windows.
Window IDs identify RPC callers; context-menu callbacks have no caller-window ID
in the pinned public callback API and must not infer one from current focus.

Interactive pickers, messages and export opt into a five-minute bridge wait;
ordinary calls retain the 30-second default. Earlier caller cancellation wins,
and cancelling a wait cannot force an OS dialog closed.

The [manual verification record](../../../docs/plans/v6-wails-windows-api-manual.md)
distinguishes observed results from unverified cases. Bulk text entry duplication
was observed and remains under investigation; do not treat IME/paste as certified.

## Repeatable manual run

1. Right-click the marked target. Select the command, toggle the checkbox, choose
   both radios in turn, open the submenu, inspect the disabled command, and dismiss
   with Escape. Observe the native menu and the corresponding report entries.
2. Open each picker. Select only the supplied fixtures, including the Unicode
   filename and multiple files. Repeat with Cancel. Save a new path and confirm
   no file was written. Open a picker from the second window to check ownership.
3. Dismiss the information dialog; choose Yes and No in the question.
   Confirm focus and normal app input recover.
4. Type synthetic text in the editable input, use keyboard selection and inspect
   its standard context menu. Test F8 and Ctrl+Shift+K, then window state controls.
5. Only if replacing the current clipboard is acceptable, explicitly write the
   fixture and check its match. Enumerate screens and compare the display details.
6. Record visual verdicts with notes, export to a new JSON file, and close using
   the title bar. Confirm the host exits after its last window closes.

Automated smoke deliberately does not click native dialogs or alter clipboard
state. It separately proves tester routing, native window RPC and session-report
delivery in the real WebView. Unit tests cover contracts, bounds and menu structure.

The pinned Windows message implementation uses fixed `MB_OK` and `MB_YESNO`
dialogs, not arbitrary custom button text. Its callback labels are exactly `Ok`,
`Yes` and `No`. A No response is a completed negative answer, not picker cancellation.

The pinned source is authoritative. Current upstream references for orientation:
[context menus](https://v3.wails.io/features/menus/context/),
[dialogs](https://v3.wails.io/reference/dialogs/), and
[menus](https://v3.wails.io/reference/menu/).
