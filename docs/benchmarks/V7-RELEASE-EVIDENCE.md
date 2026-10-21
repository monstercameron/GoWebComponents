# v7 fair production benchmark evidence

The v7 release compares the last v6 source at
`1e61695cda00a659faf8215d6a496342f75e8298` with the v7 candidate on a
dedicated GitHub Actions runner. Each lane measures v6 and v7 sequentially on
the same runner, with five samples, one warm-up, seed 6001, and the
`production` build tag. The historical v6 benchmark loops are normalized to
`b.Loop()` before measurement; the workflow saves the normalization audit with
the raw suites and runner identity.

| Lane | Run | v7 source | Status |
| --- | --- | --- | --- |
| Native amd64 | [39561403780](https://github.com/monstercameron/GoWebComponents/actions/runs/39561403780) | `a1d6455f` | Passed; reviewed below |
| Wasm | [39563464806](https://github.com/monstercameron/GoWebComponents/actions/runs/39563464806) | `d00ee361` | Failed one 15% gate entry; follow-up in progress |
| Native amd64 rerun | [39570105448](https://github.com/monstercameron/GoWebComponents/actions/runs/39570105448) | `a531cc12` | Passed; 179 matched entries, 2.55% faster geomean |
| Wasm rerun | [39570012196](https://github.com/monstercameron/GoWebComponents/actions/runs/39570012196) | `a531cc12` | Failed two 15% gate entries; follow-up in progress |
| Wasm interop fix | [39577827721](https://github.com/monstercameron/GoWebComponents/actions/runs/39577827721) | `5766ead3` | Failed only the transition-list entry; 0.17% slower geomean |
| Wasm DOM-ref fix | [39581427917](https://github.com/monstercameron/GoWebComponents/actions/runs/39581427917) | `a613ff0c` | Failed two 15% entries; 0.31% slower geomean |
| Final-runtime Wasm | [39588432681](https://github.com/monstercameron/GoWebComponents/actions/runs/39588432681) | `6a5bc346` | Failed one 15% entry; 0.12% faster geomean |
| Final-runtime native amd64 | [39588436915](https://github.com/monstercameron/GoWebComponents/actions/runs/39588436915) | `6a5bc346` | Passed; 179 matched entries, 6.11% faster geomean |

The commit-helper optimization at `a531cc12` changes a runtime benchmark
path, so both lanes were rerun on that exact source. A later interop fix and
any runtime change require a fresh Wasm comparison before release.
The Wasm run uses the corrected sub-benchmark name parser, which preserves
numeric labels such as `keys-16` and accounts for duplicate IDs by occurrence.

## Native result

Both suites finished successfully with 179 matched entries and no unmatched
entries. The v7 geometric mean is **2.13% faster** in ns/op. The comparison
classified 37 entries as improved, 59 as regressed, and 83 as noise. Raw
suite JSON, the comparison, source identity, and v6 normalization audit are
preserved in [`v7-fair/`](v7-fair/).

The largest repeated native costs deserve explicit disposition:

| Benchmark | v6 → v7 median | Change | Disposition |
| --- | ---: | ---: | --- |
| `devtools/BenchmarkExportSupportDiagnosticBundleJSON` | 63,214 → 75,114 ns | +18.8% | Accepted cost of expanded support-data redaction; +21.8% in the earlier independent run. |
| `devtools/BenchmarkSanitizeBugCaptureBundleForSupport` | 50,308 → 63,177 ns | +25.6% | Accepted redaction cost; +28.0% in the earlier run. |
| `diagnostics/BenchmarkNewReport` | 10,798 → 15,046 ns | +39.3% | Known diagnostic construction cost; +43.0% in the earlier run. Follow-up optimization is warranted, but this is an error-report path. |
| `internal/runtime/BenchmarkPerformUnitOfWorkFunctionComponentLeaf` | 365.2 → 450.6 ns | +23.4% | Runtime cost to track after release; +10.3% in the earlier run. A safe isolated fix has not been established. |
| `internal/runtime/BenchmarkPerformUnitOfWorkFunctionComponentNoPropsLeaf` | 365.9 → 442.3 ns | +20.9% | Same follow-up; +10.9% in the earlier run. |

The sub-4 ns `PropsEqualChildrenSamePointer` entry moved from flat in the
earlier run to +26.5% here; this is too small and unstable to interpret as a
product-level slowdown. The native result supports release with the listed
costs documented; the Wasm lane remains a separate gate.

The [PR benchmark drift job](https://github.com/monstercameron/GoWebComponents/actions/runs/39589049473)
also passed on `5a2ce15b`. It ran the base and head production suites
sequentially on one runner and treated the `/v6` to `/v7` module change as an
advisory major-upgrade comparison, preserving its full artifact for review.

## First corrected Wasm result

Both suites completed with all 148 entries matched. The v7 geometric mean was
1.05% slower, with 17 improved, 54 regressed, and 77 noise entries. One entry
exceeded the workflow's 15% limit: `BenchmarkTransitionListRefresh250` rose
from 1,235,680 to 1,549,813 ns/op (+25.4%). The five samples on each side
were tightly grouped, and an earlier run showed the same direction. This is a
release blocker pending a focused fix and a new dedicated runner comparison.
The artifact and full comparison are attached to the [Wasm run](https://github.com/monstercameron/GoWebComponents/actions/runs/39563464806).

## Exact-head rerun at `a531cc12`

The dedicated [native rerun](https://github.com/monstercameron/GoWebComponents/actions/runs/39570105448)
passed with 179 matched entries and a 2.55% faster v7 geometric mean. Its
runtime and diagnostic outliers are similar to those documented above.

The dedicated [Wasm rerun](https://github.com/monstercameron/GoWebComponents/actions/runs/39570012196)
completed both suites with 148 matched entries and a 0.89% slower v7 geometric
mean. It failed the 15% per-entry gate on two measurements:

| Benchmark | v6 → v7 median | Change |
| --- | ---: | ---: |
| `internal/runtime/BenchmarkTransitionListRefresh250` | 1,291,877 → 1,562,174 ns | +20.92% |
| `interop/BenchmarkDecodeMapToStruct` | 17,314 → 20,205 ns | +16.70% |

The interop entry was also +14.35% in the earlier corrected Wasm run. Both
entries remain release blockers until an exact-source dedicated run passes or
the measured costs receive an explicit release disposition.

## Interop fix checkpoint at `5766ead3`

The dedicated [Wasm comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39577827721)
completed both suites with all 148 entries matched and a 0.17% slower v7
geometric mean. `BenchmarkDecodeMapToStruct` improved from 16,987 to 7,622
ns/op (−55.13%). `BenchmarkTransitionListRefresh250` remained above the gate:
1,283,920 to 1,528,200 ns/op (+19.03%). A DOM-ref commit-path optimization and
final decoding compatibility guard landed after this measurement; dedicated
native and Wasm comparisons on the final source are still required.

## DOM-ref fix checkpoint at `a613ff0c`

The dedicated [Wasm comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39581427917)
matched all 148 entries with a 0.31% slower v7 geometric mean. The transition
list improved relative to earlier v7 runs, but remains above the 15% release
gate: 991,153 to 1,158,482 ns/op (+16.88%). The v6 samples ranged from
937,863 to 1,139,010 ns/op; the v7 samples ranged from 1,021,232 to 1,165,729
ns/op. `PropsEqualChildrenSamePointer` also crossed the initial percentage
gate, 7.103 to 10.4 ns/op (+3.297 ns, +46.42%). This benchmark and its
immediate `propsEqual`/`samePropsMap` implementation are unchanged from v6.
The established PR drift policy gates timing entries at 100 ns/op or greater,
while retaining smaller entries in the report. The fair Wasm workflow now
applies that same floor; re-evaluating this artifact leaves only the
transition-list regression as a release blocker. A fresh run is still required
for the later scheduler change.

## Final-runtime native result at `6a5bc346`

The dedicated [native comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39588436915)
completed both production suites, matched all 179 entries, and passed. The
v7 geometric mean was 6.11% faster, with 76 improved, 33 regressed, and 70
noise entries. Four entries above 100 ns/op rose by at least 15%: two
diagnostic-report constructors (8.5–8.8 to 11.1–11.3 us), support-bundle
redaction (38.2 to 44.6 us), and `RenderPortalToSelector` (14.2 to 16.4 us).
The first three are diagnostic/support paths already identified in earlier
comparisons. The portal entry was only 2.3% slower in the previous dedicated
native run, so its latest 15.6% delta is not established as a repeatable
runtime regression. Keep it under the native drift watch after release.

## Final-runtime Wasm result at `6a5bc346`

The dedicated [Wasm comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39588432681)
matched all 148 entries with a 0.12% faster v7 geometric mean. After applying
the established 100 ns/op timing floor, only
`internal/runtime/BenchmarkTransitionListRefresh250` breached the 15% gate:
the median rose from 1,240,555 to 1,522,376 ns/op (+22.72%). Its five v6
samples ranged from 1,231,314 to 1,272,773 ns/op and its five v7 samples
from 1,521,321 to 1,546,392 ns/op. Both sides allocated 1,286 objects per
operation, with nearly identical bytes. This is a repeatable CPU-path cost;
the maintainer accepted this measured v7 cost on 2026-10-21 after reviewing
the +22.72% final-run result. The prior
sub-100 ns `PropsEqualChildrenSamePointer` control was effectively flat on
this run (8.733 to 8.732 ns/op).

## Production Wasm size checkpoint

Clean v6 source at `1e61695c` and v7 source at `5a2ce15b` were built on the
same Windows arm64 host with Go 1.27.1, `GOOS=js`, `GOARCH=wasm`,
`GOEXPERIMENT=nojsonv2`, `-tags production -trimpath -ldflags "-s -w"`.
The same gzip and Brotli probe measured each paired artifact:

| Application | v6 gzip → v7 gzip | Gzip change | v6 Brotli → v7 Brotli | Brotli change |
| --- | ---: | ---: | ---: | ---: |
| Counter | 1,744,179 → 1,610,084 B | −7.69% | 1,261,105 → 1,176,789 B | −6.69% |
| Mid app | 1,738,499 → 1,605,277 B | −7.66% | 1,259,675 → 1,168,844 B | −7.21% |
| Atlas client | 4,768,854 → 4,758,449 B | −0.22% | 3,301,803 → 3,329,365 B | +0.83% |

The historical 1,600,000-byte gzip target remains missed by 10,084 B for the
counter, 5,277 B for the mid app, and 3,158,449 B for Atlas. The test logs a
miss instead of failing. Counter and mid app are materially smaller than v6;
Atlas is approximately flat and needs a separate size-reduction project. The
raw local logs are in ignored `bin/p0-evidence-5a2ce15b-clean/`. The earlier
dirty-worktree v7 size measurements were discarded.

## Release validation checkpoint

The [validation-only release run](https://github.com/monstercameron/GoWebComponents/actions/runs/39587098871)
passed its Windows `build-and-test` job at `24970ca5`; the publishing job was
skipped as requested. This covered the release workflow's root and nested
module tests, reproducible Wasm build, vulnerability scan, SBOM, changelog
entry, `gwc verify -audit`, browser smoke, examples browser suite, browser
benchmark suite, and native/Wasm candidate consumer build. Later runtime and
benchmark-harness commits required a final validation-only rerun. That
[rerun](https://github.com/monstercameron/GoWebComponents/actions/runs/39588727079)
passed its Windows `build-and-test` job on the final production runtime at
`381fc033`; publishing was skipped by `validate_only`.

## Example 201 browser checkpoint

The first dedicated [browser comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39585545329)
used the exact React/ReactDOM 19.2.4 bundle and matched all 19 scenarios, but
one `primitive-render` v7 sample took 23.5 ms while its other six samples were
1.7–3.2 ms. Its mean-based report therefore exaggerated that scenario. The
release gate was changed to use three sequential v6/v7 pairs, taking each
capture's raw-sample median and then the median across captures.

The first [three-pair run](https://github.com/monstercameron/GoWebComponents/actions/runs/39587513666)
found no scenario over the stated +30% noise limit, but v7's 19-scenario
geometric mean was 1.61% slower, so the positive-geomean exit gate remains
open. The [rerun after the scheduler fast path](https://github.com/monstercameron/GoWebComponents/actions/runs/39588530531)
also matched 19 scenarios and found no scenario over +30%; its geomean was
1.70% slower. Several captures varied by 50–100% within a side, and the phase
data did not isolate a repeatable v7 cost. The positive-geomean exit gate
remains open. The [exact-head `5a2ce15b` comparison](https://github.com/monstercameron/GoWebComponents/actions/runs/39589040373)
again matched all 19 scenarios with none above +30%, but its geomean was
4.27% slower. Individual pair geomean deltas were +1.43%, +1.38%, and +9.03%,
which shows material run variation. There is no isolated scenario suitable
for a targeted fix yet. The maintainer accepted the missed positive-geomean
target for v7 on 2026-10-21 after reviewing the 4.27% final-run result and
the two earlier three-pair comparisons. This is an explicit exception to the
V7-P3 performance exit gate; the per-scenario +30% limit was met.

## P0 browser and runtime health checkpoint

The [P0 health run](https://github.com/monstercameron/GoWebComponents/actions/runs/39588530650)
passed both jobs at `381fc033` (the same production runtime as `6a5bc346`).
Atlas cold/warm hydration, CPU profiles, shell stability, steady-state profiles,
interactions, and comments-inbox shape all passed in Chromium. The v5 load
harness passed long-frame attribution, dose response, GC-pause sweep and floor,
frame-budget, and window-transition probes. The Atlas interaction report
recorded zero frames over 50 ms. The GC floor's worst forced-collection pause
was 0.40 ms, below its 3.00 ms budget. Both raw test JSONL artifacts are
attached to the run.

## External consumer acceptance

An isolated CashFlux `/v7` migration used a clean archive of exact GWC commit
`5a2ce15b` with its pinned submodules restored. Its Wasm build and full short
Go suite passed. The production Playwright gate exited successfully with 27
passed, 2 skipped, and 2 timing-sensitive retries. CashFlux's eight generated
icon files needed regeneration under Go 1.27.1: the original v5 application
source and assets pass that fixture test under Go 1.26 but fail unchanged
under Go 1.27.1, so this is a toolchain output change rather than a v7
framework regression. The migration and regenerated files remain in the
isolated acceptance checkout, not in the CashFlux source repository. Logs are
saved under `cashflux-v7-acceptance-20261021/bin/` in the task workspace.

The pinned GoGRPCBridge native stress and 10-client browser stress runs passed
against the v7 source. The browser harness used a temporary older Playwright
Go driver because the exact v0.6000 browser-driver download returned 404; the
bridge source and pin were unchanged. Its log is in ignored
`bin/test-results/v7-gogrpcbridge-stress-20261021.txt`.

## Release candidate and external module fetch

The [`v7.0.0-rc.1` release workflow](https://github.com/monstercameron/GoWebComponents/actions/runs/39595468422)
passed both the Windows `build-and-test` and publishing jobs. Its tag points
to `317bf88b71911ea7ed39ad710d40ca45a566d80a`, which adds only this
release evidence and notes after the measured runtime commit. The
[GitHub prerelease](https://github.com/monstercameron/GoWebComponents/releases/tag/v7.0.0-rc.1)
contains the source archive, checksums, CycloneDX SBOM, and six platform
`gwc` binaries.

A fresh module fetched
`github.com/monstercameron/GoWebComponents/v7@v7.0.0-rc.1` through the Go
module proxy (sum `h1:RnU+Cd1BQ5e9NoW89CP6e2t2lsAaqTHF9eGSMZriGDo=`),
ran `go mod tidy`, and compiled imports of `ui`, `html`, and `router` both
natively and with `GOOS=js GOARCH=wasm GOEXPERIMENT=nojsonv2`. The isolated
consumer files are in ignored `bin/v7-rc-consumer-20261021/`.
