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

The native benchmark packages did not change between those v7 commits.
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

## First corrected Wasm result

Both suites completed with all 148 entries matched. The v7 geometric mean was
1.05% slower, with 17 improved, 54 regressed, and 77 noise entries. One entry
exceeded the workflow's 15% limit: `BenchmarkTransitionListRefresh250` rose
from 1,235,680 to 1,549,813 ns/op (+25.4%). The five samples on each side
were tightly grouped, and an earlier run showed the same direction. This is a
release blocker pending a focused fix and a new dedicated runner comparison.
The artifact and full comparison are attached to the [Wasm run](https://github.com/monstercameron/GoWebComponents/actions/runs/39563464806).
