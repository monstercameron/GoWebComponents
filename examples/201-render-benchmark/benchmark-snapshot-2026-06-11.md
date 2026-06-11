# Example 201 Browser Benchmark Report

- Generated at: `2026-06-11T17:51:10Z`
- Browser: `chromium`
- Route: `/examples/testing/render-benchmark/?iterations=7&warmups=2&seed=20101`
- Iterations: `7`
- Warmups: `2`
- Seed: `20101`
- Scenario order: `primitive-remove, core-append, deep-render, content-update, content-render, primitive-text-update, enterprise-subtree-update, primitive-append, core-update, core-refresh, deep-update, deep-refresh, content-refresh, core-filter, hooks-render, primitive-attribute-update, core-stress-update, primitive-render, core-render`
- Framework order: `runtime1, runtime2-workers2, runtime2-workers4, runtime2-workers8, runtime2-workers1, react`
- Fairness note: framework order rotates per scenario, so no framework keeps the same warm-cache or JIT slot across the whole run.
- Important boundary: the `runtime2` subjects here still keep DOM ownership on the main thread.
- Worker note: each `Runtime 2 (N Workers)` subject opens the requested Go WASM worker count to prepare core and content chunks before the local runtime2 shell commits DOM updates.
- Non-worker note: deep-tree, primitive, hook-grid, and enterprise-workspace subtree scenarios remain main-thread-owned today, so the worker-backed benefit is expected to concentrate in the core and content scenarios.
- React subject note: the page uses a vendored React 19.2.4 browser bundle under `examples/testing/render-benchmark/vendor/`, so the comparison stays local to the repo server.

- Finish lines: `DOM Ready` means the scenario correctness contract became true. `Paint Proxy` means one `requestAnimationFrame` boundary after the DOM-ready checkpoint.
- Primary comparison: category summaries and scenario ordering use `DOM Ready` as the lead timing. `Paint Proxy` stays in the report as secondary frame-bound context only.
- Score reference: `local` on `chromium` from route `/examples/testing/render-benchmark/`. `DOM Score` is `100 * geometric_mean(reference DOM Ready / measured DOM Ready)`.
- Mixed-framework score direction: higher `DOM Score` is faster than the fixed reference profile. `DOM vs React` stays as a separate same-run diagnostic column.
- Worker diagnostics: worker-backed subjects also report whether the measured run triggered chunk preparation, the mean batch count, the last-batch duration, and the prepared-item count for that run window.
- RT2 scaling view: the dedicated scaling section compares worker-preparation batch time first, because paint-proxy is often frame-quantized and can hide real worker-count differences.
- RT2 stress route: `/examples/testing/render-benchmark/?iterations=5&runtime2Dispatch=batch&runtime2WorkScale=12&runtime2WorkerCounts=1%2C2%2C4%2C8&seed=20101&subjectSet=runtime2-scaling&warmups=1` reruns RT2-only scaling with heavier worker prep so the end-to-end timing spreads beyond one frame when possible.
- Headline scope note: worker-relevant overall score excludes deep-tree, primitive, hook-grid, and enterprise-workspace subtree scenarios because those paths are still main-thread-owned in runtime2 today.

## Overall DOM Score (Worker-Relevant)

- Score contract: `100` equals the checked-in reference profile for the worker-relevant scenarios only.

| Framework | DOM Score | Geom. DOM Score Factor | Scored Scenarios | Total Scenarios |
| --- | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 1.000x | 0 | 19 |
| Runtime 1 | 0 | 1.000x | 0 | 19 |
| Runtime 2 (1 Worker) | 0 | 1.000x | 0 | 19 |
| Runtime 2 (2 Workers) | 0 | 1.000x | 0 | 19 |
| Runtime 2 (4 Workers) | 0 | 1.000x | 0 | 19 |
| Runtime 2 (8 Workers) | 0 | 1.000x | 0 | 19 |

## Overall DOM Score (Full Surface)

- Full-surface score includes every active scenario, including primitive, deep-tree, hook-grid, and enterprise-workspace subtree paths.

| Framework | DOM Score | Geom. DOM Score Factor | Scored Scenarios | Total Scenarios |
| --- | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 1.000x | 0 | 19 |
| Runtime 1 | 0 | 1.000x | 0 | 19 |
| Runtime 2 (1 Worker) | 0 | 1.000x | 0 | 19 |
| Runtime 2 (2 Workers) | 0 | 1.000x | 0 | 19 |
| Runtime 2 (4 Workers) | 0 | 1.000x | 0 | 19 |
| Runtime 2 (8 Workers) | 0 | 1.000x | 0 | 19 |

## Category Summary

| Category | Framework | DOM Score | Avg DOM Ready | Avg DOM vs React | Avg Paint Proxy | Geom. DOM Score Factor | Wins |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Initial Render | React 19.2.4 | 0 | 2.414 ms | +0.000 ms | 19.200 ms | n/a | 4 |
| Initial Render | Runtime 1 | 0 | 4.893 ms | -2.479 ms | 13.303 ms | n/a | 0 |
| Initial Render | Runtime 2 (1 Worker) | 0 | 10.511 ms | -8.097 ms | 15.997 ms | n/a | 0 |
| Initial Render | Runtime 2 (2 Workers) | 0 | 12.211 ms | -9.797 ms | 18.311 ms | n/a | 0 |
| Initial Render | Runtime 2 (4 Workers) | 0 | 12.211 ms | -9.797 ms | 18.325 ms | n/a | 0 |
| Initial Render | Runtime 2 (8 Workers) | 0 | 11.839 ms | -9.425 ms | 18.147 ms | n/a | 0 |
| Primitive Churn | React 19.2.4 | 0 | 2.671 ms | +0.000 ms | 19.922 ms | n/a | 2 |
| Primitive Churn | Runtime 1 | 0 | 9.864 ms | -7.193 ms | 14.835 ms | n/a | 0 |
| Primitive Churn | Runtime 2 (1 Worker) | 0 | 4.307 ms | -1.635 ms | 14.072 ms | n/a | 0 |
| Primitive Churn | Runtime 2 (2 Workers) | 0 | 6.400 ms | -3.728 ms | 16.357 ms | n/a | 0 |
| Primitive Churn | Runtime 2 (4 Workers) | 0 | 5.329 ms | -2.657 ms | 16.421 ms | n/a | 0 |
| Primitive Churn | Runtime 2 (8 Workers) | 0 | 4.657 ms | -1.986 ms | 16.215 ms | n/a | 0 |
| Primitive Render | React 19.2.4 | 0 | 3.000 ms | +0.000 ms | 20.829 ms | n/a | 1 |
| Primitive Render | Runtime 1 | 0 | 6.100 ms | -3.100 ms | 12.129 ms | n/a | 0 |
| Primitive Render | Runtime 2 (1 Worker) | 0 | 5.257 ms | -2.257 ms | 15.157 ms | n/a | 0 |
| Primitive Render | Runtime 2 (2 Workers) | 0 | 8.043 ms | -5.043 ms | 16.414 ms | n/a | 0 |
| Primitive Render | Runtime 2 (4 Workers) | 0 | 7.257 ms | -4.257 ms | 16.686 ms | n/a | 0 |
| Primitive Render | Runtime 2 (8 Workers) | 0 | 5.214 ms | -2.214 ms | 16.500 ms | n/a | 0 |
| Primitive Update | React 19.2.4 | 0 | 2.543 ms | +0.000 ms | 17.907 ms | n/a | 2 |
| Primitive Update | Runtime 1 | 0 | 4.686 ms | -2.143 ms | 16.750 ms | n/a | 0 |
| Primitive Update | Runtime 2 (1 Worker) | 0 | 5.614 ms | -3.072 ms | 16.650 ms | n/a | 0 |
| Primitive Update | Runtime 2 (2 Workers) | 0 | 6.014 ms | -3.471 ms | 16.386 ms | n/a | 0 |
| Primitive Update | Runtime 2 (4 Workers) | 0 | 5.364 ms | -2.822 ms | 16.407 ms | n/a | 0 |
| Primitive Update | Runtime 2 (8 Workers) | 0 | 6.314 ms | -3.771 ms | 16.550 ms | n/a | 0 |
| Refresh | React 19.2.4 | 0 | 1.719 ms | +0.000 ms | 16.591 ms | n/a | 3 |
| Refresh | Runtime 1 | 0 | 3.895 ms | -2.176 ms | 16.333 ms | n/a | 0 |
| Refresh | Runtime 2 (1 Worker) | 0 | 5.734 ms | -4.015 ms | 16.614 ms | n/a | 0 |
| Refresh | Runtime 2 (2 Workers) | 0 | 6.000 ms | -4.281 ms | 16.295 ms | n/a | 0 |
| Refresh | Runtime 2 (4 Workers) | 0 | 5.957 ms | -4.238 ms | 16.262 ms | n/a | 0 |
| Refresh | Runtime 2 (8 Workers) | 0 | 6.381 ms | -4.662 ms | 16.562 ms | n/a | 0 |
| Structural Churn | React 19.2.4 | 0 | 2.979 ms | +0.000 ms | 20.129 ms | n/a | 2 |
| Structural Churn | Runtime 1 | 0 | 7.143 ms | -4.164 ms | 10.308 ms | n/a | 0 |
| Structural Churn | Runtime 2 (1 Worker) | 0 | 28.722 ms | -25.742 ms | 30.042 ms | n/a | 0 |
| Structural Churn | Runtime 2 (2 Workers) | 0 | 36.786 ms | -33.807 ms | 38.843 ms | n/a | 0 |
| Structural Churn | Runtime 2 (4 Workers) | 0 | 31.914 ms | -28.935 ms | 32.857 ms | n/a | 0 |
| Structural Churn | Runtime 2 (8 Workers) | 0 | 31.971 ms | -28.992 ms | 32.978 ms | n/a | 0 |
| Targeted Update | React 19.2.4 | 0 | 2.737 ms | +0.000 ms | 17.626 ms | n/a | 5 |
| Targeted Update | Runtime 1 | 0 | 4.246 ms | -1.509 ms | 16.632 ms | n/a | 0 |
| Targeted Update | Runtime 2 (1 Worker) | 0 | 9.911 ms | -7.174 ms | 18.020 ms | n/a | 0 |
| Targeted Update | Runtime 2 (2 Workers) | 0 | 10.200 ms | -7.463 ms | 18.443 ms | n/a | 0 |
| Targeted Update | Runtime 2 (4 Workers) | 0 | 11.372 ms | -8.635 ms | 18.428 ms | n/a | 0 |
| Targeted Update | Runtime 2 (8 Workers) | 0 | 10.606 ms | -7.869 ms | 17.457 ms | n/a | 0 |

## RT2 Worker Scaling

### Core Render

- Category: `Initial Render`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 5.157 ms | 1.000x | 0.057 ms | 1.000x | 40.0 | 15.257 ms |
| Runtime 2 (2 Workers) | 7.571 ms | 0.681x | 0.071 ms | 0.803x | 40.0 | 16.786 ms |
| Runtime 2 (4 Workers) | 7.929 ms | 0.650x | 0.157 ms | 0.363x | 40.0 | 17.157 ms |
| Runtime 2 (8 Workers) | 5.071 ms | 1.017x | 0.071 ms | 0.803x | 40.0 | 16.729 ms |

### Core Update

- Category: `Targeted Update`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 3.957 ms | 1.000x | 0.086 ms | 1.000x | 40.0 | 16.200 ms |
| Runtime 2 (2 Workers) | 7.800 ms | 0.507x | 0.071 ms | 1.211x | 40.0 | 17.029 ms |
| Runtime 2 (4 Workers) | 5.157 ms | 0.767x | 0.100 ms | 0.860x | 40.0 | 17.200 ms |
| Runtime 2 (8 Workers) | 6.100 ms | 0.649x | 0.100 ms | 0.860x | 40.0 | 16.429 ms |

### Core Stress Update

- Category: `Targeted Update`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 23.757 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 23.800 ms |
| Runtime 2 (2 Workers) | 24.129 ms | 0.985x | 0.000 ms | 1.000x | 0.0 | 24.200 ms |
| Runtime 2 (4 Workers) | 24.829 ms | 0.957x | 0.000 ms | 1.000x | 0.0 | 24.871 ms |
| Runtime 2 (8 Workers) | 20.400 ms | 1.165x | 0.000 ms | 1.000x | 0.0 | 20.400 ms |

### Core Refresh

- Category: `Refresh`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 2.529 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.643 ms |
| Runtime 2 (2 Workers) | 3.571 ms | 0.708x | 0.000 ms | 1.000x | 0.0 | 16.229 ms |
| Runtime 2 (4 Workers) | 2.986 ms | 0.847x | 0.000 ms | 1.000x | 0.0 | 16.271 ms |
| Runtime 2 (8 Workers) | 3.386 ms | 0.747x | 0.000 ms | 1.000x | 0.0 | 16.314 ms |

### Core Append

- Category: `Structural Churn`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 36.843 ms | 1.000x | 21.657 ms | 1.000x | 340.0 | 37.471 ms |
| Runtime 2 (2 Workers) | 53.943 ms | 0.683x | 26.900 ms | 0.805x | 340.0 | 57.129 ms |
| Runtime 2 (4 Workers) | 47.800 ms | 0.771x | 23.514 ms | 0.921x | 340.0 | 47.900 ms |
| Runtime 2 (8 Workers) | 46.243 ms | 0.797x | 22.457 ms | 0.964x | 340.0 | 46.286 ms |

### Core Filter

- Category: `Structural Churn`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 20.600 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 22.614 ms |
| Runtime 2 (2 Workers) | 19.629 ms | 1.049x | 0.000 ms | 1.000x | 0.0 | 20.557 ms |
| Runtime 2 (4 Workers) | 16.029 ms | 1.285x | 0.000 ms | 1.000x | 0.0 | 17.814 ms |
| Runtime 2 (8 Workers) | 17.700 ms | 1.164x | 0.000 ms | 1.000x | 0.0 | 19.671 ms |

### Content Render

- Category: `Initial Render`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 12.329 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 13.657 ms |
| Runtime 2 (2 Workers) | 11.229 ms | 1.098x | 0.000 ms | 1.000x | 0.0 | 17.014 ms |
| Runtime 2 (4 Workers) | 13.343 ms | 0.924x | 0.000 ms | 1.000x | 0.0 | 17.771 ms |
| Runtime 2 (8 Workers) | 12.200 ms | 1.011x | 0.000 ms | 1.000x | 0.0 | 16.743 ms |

### Content Update

- Category: `Targeted Update`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 8.357 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.771 ms |
| Runtime 2 (2 Workers) | 9.171 ms | 0.911x | 0.000 ms | 1.000x | 0.0 | 17.414 ms |
| Runtime 2 (4 Workers) | 9.629 ms | 0.868x | 0.000 ms | 1.000x | 0.0 | 16.386 ms |
| Runtime 2 (8 Workers) | 8.771 ms | 0.953x | 0.000 ms | 1.000x | 0.0 | 16.729 ms |

### Content Refresh

- Category: `Refresh`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 7.329 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.929 ms |
| Runtime 2 (2 Workers) | 8.829 ms | 0.830x | 0.000 ms | 1.000x | 0.0 | 16.400 ms |
| Runtime 2 (4 Workers) | 8.386 ms | 0.874x | 0.000 ms | 1.000x | 0.0 | 16.486 ms |
| Runtime 2 (8 Workers) | 9.157 ms | 0.800x | 0.000 ms | 1.000x | 0.0 | 17.057 ms |

### Primitive Render

- Category: `Primitive Render`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 5.257 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 15.157 ms |
| Runtime 2 (2 Workers) | 8.043 ms | 0.654x | 0.000 ms | 1.000x | 0.0 | 16.414 ms |
| Runtime 2 (4 Workers) | 7.257 ms | 0.724x | 0.000 ms | 1.000x | 0.0 | 16.686 ms |
| Runtime 2 (8 Workers) | 5.214 ms | 1.008x | 0.000 ms | 1.000x | 0.0 | 16.500 ms |

### Primitive Text Update

- Category: `Primitive Update`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 4.086 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.700 ms |
| Runtime 2 (2 Workers) | 5.414 ms | 0.755x | 0.000 ms | 1.000x | 0.0 | 16.443 ms |
| Runtime 2 (4 Workers) | 5.800 ms | 0.704x | 0.000 ms | 1.000x | 0.0 | 16.443 ms |
| Runtime 2 (8 Workers) | 6.171 ms | 0.662x | 0.000 ms | 1.000x | 0.0 | 16.543 ms |

### Primitive Attribute Update

- Category: `Primitive Update`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 7.143 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.600 ms |
| Runtime 2 (2 Workers) | 6.614 ms | 1.080x | 0.000 ms | 1.000x | 0.0 | 16.329 ms |
| Runtime 2 (4 Workers) | 4.929 ms | 1.449x | 0.000 ms | 1.000x | 0.0 | 16.371 ms |
| Runtime 2 (8 Workers) | 6.457 ms | 1.106x | 0.000 ms | 1.000x | 0.0 | 16.557 ms |

### Primitive Append

- Category: `Primitive Churn`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 4.971 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 15.143 ms |
| Runtime 2 (2 Workers) | 7.400 ms | 0.672x | 0.000 ms | 1.000x | 0.0 | 16.700 ms |
| Runtime 2 (4 Workers) | 5.357 ms | 0.928x | 0.000 ms | 1.000x | 0.0 | 16.471 ms |
| Runtime 2 (8 Workers) | 5.400 ms | 0.921x | 0.000 ms | 1.000x | 0.0 | 15.986 ms |

### Primitive Remove

- Category: `Primitive Churn`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 3.643 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 13.000 ms |
| Runtime 2 (2 Workers) | 5.400 ms | 0.675x | 0.000 ms | 1.000x | 0.0 | 16.014 ms |
| Runtime 2 (4 Workers) | 5.300 ms | 0.687x | 0.000 ms | 1.000x | 0.0 | 16.371 ms |
| Runtime 2 (8 Workers) | 3.914 ms | 0.931x | 0.000 ms | 1.000x | 0.0 | 16.443 ms |

### Deep Tree Render

- Category: `Initial Render`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 8.529 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.729 ms |
| Runtime 2 (2 Workers) | 9.557 ms | 0.892x | 0.000 ms | 1.000x | 0.0 | 17.014 ms |
| Runtime 2 (4 Workers) | 6.300 ms | 1.354x | 0.000 ms | 1.000x | 0.0 | 17.071 ms |
| Runtime 2 (8 Workers) | 9.514 ms | 0.896x | 0.000 ms | 1.000x | 0.0 | 16.786 ms |

### Deep Tree Update

- Category: `Targeted Update`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 7.586 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.814 ms |
| Runtime 2 (2 Workers) | 4.929 ms | 1.539x | 0.000 ms | 1.000x | 0.0 | 16.657 ms |
| Runtime 2 (4 Workers) | 11.229 ms | 0.676x | 0.000 ms | 1.000x | 0.0 | 17.414 ms |
| Runtime 2 (8 Workers) | 10.743 ms | 0.706x | 0.000 ms | 1.000x | 0.0 | 16.786 ms |

### Deep Tree Refresh

- Category: `Refresh`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 7.343 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.271 ms |
| Runtime 2 (2 Workers) | 5.600 ms | 1.311x | 0.000 ms | 1.000x | 0.0 | 16.257 ms |
| Runtime 2 (4 Workers) | 6.500 ms | 1.130x | 0.000 ms | 1.000x | 0.0 | 16.029 ms |
| Runtime 2 (8 Workers) | 6.600 ms | 1.113x | 0.000 ms | 1.000x | 0.0 | 16.314 ms |

### Enterprise Subtree Update

- Category: `Targeted Update`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 5.900 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.514 ms |
| Runtime 2 (2 Workers) | 4.971 ms | 1.187x | 0.000 ms | 1.000x | 0.0 | 16.914 ms |
| Runtime 2 (4 Workers) | 6.014 ms | 0.981x | 0.000 ms | 1.000x | 0.0 | 16.271 ms |
| Runtime 2 (8 Workers) | 7.014 ms | 0.841x | 0.000 ms | 1.000x | 0.0 | 16.943 ms |

### Hook Grid Render

- Category: `Initial Render`
- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.

| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 16.029 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 18.343 ms |
| Runtime 2 (2 Workers) | 20.486 ms | 0.782x | 0.000 ms | 1.000x | 0.0 | 22.429 ms |
| Runtime 2 (4 Workers) | 21.271 ms | 0.754x | 0.000 ms | 1.000x | 0.0 | 21.300 ms |
| Runtime 2 (8 Workers) | 20.571 ms | 0.779x | 0.000 ms | 1.000x | 0.0 | 22.329 ms |


## RT2 Worker Scaling Stress

- Description: RT2-only stress run with runtime2WorkScale=12 to expose worker-count scaling beyond frame-quantized paint-proxy timing.
- Route: `/examples/testing/render-benchmark/?iterations=5&runtime2Dispatch=batch&runtime2WorkScale=12&runtime2WorkerCounts=1%2C2%2C4%2C8&seed=20101&subjectSet=runtime2-scaling&warmups=1`
- This route is RT2-only and uses a higher worker work scale so DOM-ready and paint metrics stop collapsing into the same one-frame bucket.

### Core Render

- Category: `Initial Render`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 7.420 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 15.400 ms |
| Runtime 2 (2 Workers) | 6.580 ms | 1.128x | 0.000 ms | 1.000x | 0.0 | 16.840 ms |
| Runtime 2 (4 Workers) | 5.980 ms | 1.241x | 0.000 ms | 1.000x | 0.0 | 16.420 ms |
| Runtime 2 (8 Workers) | 8.420 ms | 0.881x | 0.000 ms | 1.000x | 0.0 | 16.320 ms |

### Core Update

- Category: `Targeted Update`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 6.840 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.600 ms |
| Runtime 2 (2 Workers) | 9.380 ms | 0.729x | 0.000 ms | 1.000x | 0.0 | 16.700 ms |
| Runtime 2 (4 Workers) | 9.140 ms | 0.748x | 0.000 ms | 1.000x | 0.0 | 17.260 ms |
| Runtime 2 (8 Workers) | 9.200 ms | 0.743x | 0.000 ms | 1.000x | 0.0 | 17.120 ms |

### Core Stress Update

- Category: `Targeted Update`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 21.780 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 21.880 ms |
| Runtime 2 (2 Workers) | 23.900 ms | 0.911x | 0.000 ms | 1.000x | 0.0 | 24.000 ms |
| Runtime 2 (4 Workers) | 21.780 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 21.820 ms |
| Runtime 2 (8 Workers) | 24.300 ms | 0.896x | 0.000 ms | 1.000x | 0.0 | 24.360 ms |

### Core Refresh

- Category: `Refresh`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 1.920 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.780 ms |
| Runtime 2 (2 Workers) | 1.860 ms | 1.032x | 0.000 ms | 1.000x | 0.0 | 16.680 ms |
| Runtime 2 (4 Workers) | 1.980 ms | 0.970x | 0.000 ms | 1.000x | 0.0 | 16.480 ms |
| Runtime 2 (8 Workers) | 2.300 ms | 0.835x | 0.000 ms | 1.000x | 0.0 | 16.480 ms |

### Core Append

- Category: `Structural Churn`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 37.560 ms | 1.000x | 24.300 ms | 1.000x | 340.0 | 40.420 ms |
| Runtime 2 (2 Workers) | 45.840 ms | 0.819x | 24.440 ms | 0.994x | 340.0 | 46.680 ms |
| Runtime 2 (4 Workers) | 42.660 ms | 0.880x | 20.980 ms | 1.158x | 340.0 | 42.760 ms |
| Runtime 2 (8 Workers) | 47.020 ms | 0.799x | 26.700 ms | 0.910x | 340.0 | 47.120 ms |

### Core Filter

- Category: `Structural Churn`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 15.500 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 15.560 ms |
| Runtime 2 (2 Workers) | 18.780 ms | 0.825x | 0.000 ms | 1.000x | 0.0 | 19.080 ms |
| Runtime 2 (4 Workers) | 16.600 ms | 0.934x | 0.000 ms | 1.000x | 0.0 | 17.320 ms |
| Runtime 2 (8 Workers) | 18.820 ms | 0.824x | 0.000 ms | 1.000x | 0.0 | 19.560 ms |

### Content Render

- Category: `Initial Render`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 10.840 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.380 ms |
| Runtime 2 (2 Workers) | 9.940 ms | 1.091x | 0.000 ms | 1.000x | 0.0 | 16.400 ms |
| Runtime 2 (4 Workers) | 9.620 ms | 1.127x | 0.000 ms | 1.000x | 0.0 | 14.640 ms |
| Runtime 2 (8 Workers) | 12.340 ms | 0.878x | 0.000 ms | 1.000x | 0.0 | 16.520 ms |

### Content Update

- Category: `Targeted Update`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 7.920 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.460 ms |
| Runtime 2 (2 Workers) | 6.920 ms | 1.145x | 0.000 ms | 1.000x | 0.0 | 16.340 ms |
| Runtime 2 (4 Workers) | 5.940 ms | 1.333x | 0.000 ms | 1.000x | 0.0 | 16.640 ms |
| Runtime 2 (8 Workers) | 6.320 ms | 1.253x | 0.000 ms | 1.000x | 0.0 | 16.440 ms |

### Content Refresh

- Category: `Refresh`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 8.360 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.300 ms |
| Runtime 2 (2 Workers) | 5.560 ms | 1.504x | 0.000 ms | 1.000x | 0.0 | 16.920 ms |
| Runtime 2 (4 Workers) | 7.060 ms | 1.184x | 0.000 ms | 1.000x | 0.0 | 16.500 ms |
| Runtime 2 (8 Workers) | 6.380 ms | 1.310x | 0.000 ms | 1.000x | 0.0 | 16.380 ms |

### Primitive Render

- Category: `Primitive Render`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 5.820 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.520 ms |
| Runtime 2 (2 Workers) | 5.400 ms | 1.078x | 0.000 ms | 1.000x | 0.0 | 16.380 ms |
| Runtime 2 (4 Workers) | 6.800 ms | 0.856x | 0.000 ms | 1.000x | 0.0 | 16.820 ms |
| Runtime 2 (8 Workers) | 7.200 ms | 0.808x | 0.000 ms | 1.000x | 0.0 | 16.240 ms |

### Primitive Text Update

- Category: `Primitive Update`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 7.440 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 17.200 ms |
| Runtime 2 (2 Workers) | 4.100 ms | 1.815x | 0.000 ms | 1.000x | 0.0 | 16.480 ms |
| Runtime 2 (4 Workers) | 4.880 ms | 1.525x | 0.000 ms | 1.000x | 0.0 | 17.360 ms |
| Runtime 2 (8 Workers) | 4.900 ms | 1.518x | 0.000 ms | 1.000x | 0.0 | 16.480 ms |

### Primitive Attribute Update

- Category: `Primitive Update`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 8.980 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 17.160 ms |
| Runtime 2 (2 Workers) | 6.880 ms | 1.305x | 0.000 ms | 1.000x | 0.0 | 16.320 ms |
| Runtime 2 (4 Workers) | 6.220 ms | 1.444x | 0.000 ms | 1.000x | 0.0 | 16.000 ms |
| Runtime 2 (8 Workers) | 7.500 ms | 1.197x | 0.000 ms | 1.000x | 0.0 | 16.660 ms |

### Primitive Append

- Category: `Primitive Churn`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 5.120 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.420 ms |
| Runtime 2 (2 Workers) | 6.660 ms | 0.769x | 0.000 ms | 1.000x | 0.0 | 13.120 ms |
| Runtime 2 (4 Workers) | 5.160 ms | 0.992x | 0.000 ms | 1.000x | 0.0 | 15.300 ms |
| Runtime 2 (8 Workers) | 6.360 ms | 0.805x | 0.000 ms | 1.000x | 0.0 | 16.620 ms |

### Primitive Remove

- Category: `Primitive Churn`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 3.320 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 17.260 ms |
| Runtime 2 (2 Workers) | 2.940 ms | 1.129x | 0.000 ms | 1.000x | 0.0 | 14.800 ms |
| Runtime 2 (4 Workers) | 3.360 ms | 0.988x | 0.000 ms | 1.000x | 0.0 | 11.260 ms |
| Runtime 2 (8 Workers) | 4.280 ms | 0.776x | 0.000 ms | 1.000x | 0.0 | 16.360 ms |

### Deep Tree Render

- Category: `Initial Render`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 7.600 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.820 ms |
| Runtime 2 (2 Workers) | 6.480 ms | 1.173x | 0.000 ms | 1.000x | 0.0 | 16.500 ms |
| Runtime 2 (4 Workers) | 6.740 ms | 1.128x | 0.000 ms | 1.000x | 0.0 | 16.560 ms |
| Runtime 2 (8 Workers) | 8.680 ms | 0.876x | 0.000 ms | 1.000x | 0.0 | 16.640 ms |

### Deep Tree Update

- Category: `Targeted Update`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 8.140 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.280 ms |
| Runtime 2 (2 Workers) | 5.480 ms | 1.485x | 0.000 ms | 1.000x | 0.0 | 16.180 ms |
| Runtime 2 (4 Workers) | 6.060 ms | 1.343x | 0.000 ms | 1.000x | 0.0 | 16.000 ms |
| Runtime 2 (8 Workers) | 9.040 ms | 0.900x | 0.000 ms | 1.000x | 0.0 | 17.240 ms |

### Deep Tree Refresh

- Category: `Refresh`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 6.060 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.060 ms |
| Runtime 2 (2 Workers) | 4.700 ms | 1.289x | 0.000 ms | 1.000x | 0.0 | 16.400 ms |
| Runtime 2 (4 Workers) | 7.320 ms | 0.828x | 0.000 ms | 1.000x | 0.0 | 16.820 ms |
| Runtime 2 (8 Workers) | 5.800 ms | 1.045x | 0.000 ms | 1.000x | 0.0 | 15.760 ms |

### Enterprise Subtree Update

- Category: `Targeted Update`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 5.660 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 16.620 ms |
| Runtime 2 (2 Workers) | 4.920 ms | 1.150x | 0.000 ms | 1.000x | 0.0 | 16.600 ms |
| Runtime 2 (4 Workers) | 5.600 ms | 1.011x | 0.000 ms | 1.000x | 0.0 | 16.680 ms |
| Runtime 2 (8 Workers) | 5.320 ms | 1.064x | 0.000 ms | 1.000x | 0.0 | 16.760 ms |

### Hook Grid Render

- Category: `Initial Render`
| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Runtime 2 (1 Worker) | 15.580 ms | 1.000x | 0.000 ms | 1.000x | 0.0 | 17.840 ms |
| Runtime 2 (2 Workers) | 15.520 ms | 1.004x | 0.000 ms | 1.000x | 0.0 | 17.980 ms |
| Runtime 2 (4 Workers) | 17.640 ms | 0.883x | 0.000 ms | 1.000x | 0.0 | 19.220 ms |
| Runtime 2 (8 Workers) | 13.900 ms | 1.121x | 0.000 ms | 1.000x | 0.0 | 17.440 ms |


## Scenarios

### Core Render

- Category: `Initial Render`
- Requested work: Render 40 visible core-list rows from empty state.
- Correctness check: 40 .benchmark-core-item nodes must exist.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 1.514 ms | 1.400 ms | 18.414 ms | 18.300 ms | 16.900 ms | 40.0 / 0.0 / 0.0 | 40.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 1 | 0 | 3.429 ms | 3.100 ms | 12.657 ms | 13.000 ms | 9.229 ms | 1.0 / 0.0 / 0.0 | 40.0 / 0.0 | n/a | 0.0 / 0.000 ms | -1.915 ms / 0.442x |
| Runtime 2 (8 Workers) | 0 | 5.071 ms | 4.400 ms | 16.729 ms | 16.700 ms | 11.657 ms | 3.0 / 0.0 / 0.0 | 42.0 / 0.0 | 1.0 / 0.071 ms / 40.0 | 0.0 / 0.000 ms | -3.557 ms / 0.299x |
| Runtime 2 (1 Worker) | 0 | 5.157 ms | 4.700 ms | 15.257 ms | 15.000 ms | 10.100 ms | 3.0 / 0.0 / 0.0 | 42.0 / 0.0 | 1.0 / 0.057 ms / 40.0 | 0.0 / 0.000 ms | -3.643 ms / 0.294x |
| Runtime 2 (2 Workers) | 0 | 7.571 ms | 5.300 ms | 16.786 ms | 16.900 ms | 9.214 ms | 3.0 / 0.0 / 0.0 | 42.0 / 0.0 | 1.0 / 0.071 ms / 40.0 | 0.0 / 0.000 ms | -6.057 ms / 0.200x |
| Runtime 2 (4 Workers) | 0 | 7.929 ms | 6.700 ms | 17.157 ms | 17.200 ms | 9.229 ms | 3.0 / 0.0 / 0.0 | 42.0 / 0.0 | 1.0 / 0.157 ms / 40.0 | 0.0 / 0.000 ms | -6.415 ms / 0.191x |

### Core Update

- Category: `Targeted Update`
- Requested work: Update the visible text content of the existing 40 core-list rows.
- Correctness check: All .benchmark-core-item nodes must include '(Updated)' and item count must stay at 40.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 1.900 ms | 2.400 ms | 16.214 ms | 16.100 ms | 14.314 ms | 0.0 / 0.0 / 40.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 1 | 0 | 3.000 ms | 2.500 ms | 16.329 ms | 16.400 ms | 13.329 ms | 0.0 / 0.0 / 40.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | -1.100 ms / 0.633x |
| Runtime 2 (1 Worker) | 0 | 3.957 ms | 3.600 ms | 16.200 ms | 16.100 ms | 12.243 ms | 0.0 / 0.0 / 40.0 | 0.0 / 0.0 | 1.0 / 0.086 ms / 40.0 | 0.0 / 0.000 ms | -2.057 ms / 0.480x |
| Runtime 2 (4 Workers) | 0 | 5.157 ms | 4.800 ms | 17.200 ms | 17.000 ms | 12.043 ms | 0.0 / 0.0 / 40.0 | 0.0 / 0.0 | 1.0 / 0.100 ms / 40.0 | 0.0 / 0.000 ms | -3.257 ms / 0.368x |
| Runtime 2 (8 Workers) | 0 | 6.100 ms | 5.400 ms | 16.429 ms | 16.100 ms | 10.329 ms | 0.0 / 0.0 / 40.0 | 0.0 / 0.0 | 1.0 / 0.100 ms / 40.0 | 0.0 / 0.000 ms | -4.200 ms / 0.311x |
| Runtime 2 (2 Workers) | 0 | 7.800 ms | 5.000 ms | 17.029 ms | 17.000 ms | 9.229 ms | 0.0 / 0.0 / 40.0 | 0.0 / 0.0 | 1.0 / 0.071 ms / 40.0 | 0.0 / 0.000 ms | -5.900 ms / 0.244x |

### Core Stress Update

- Category: `Targeted Update`
- Requested work: Update the visible text content of the existing 240-row stress core list.
- Correctness check: All 240 .benchmark-core-item nodes must include '(Updated)' and row count must stay at 240.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 3.786 ms | 3.400 ms | 16.800 ms | 16.800 ms | 13.014 ms | 0.0 / 0.0 / 240.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 1 | 0 | 5.543 ms | 4.400 ms | 16.386 ms | 16.100 ms | 10.843 ms | 0.0 / 0.0 / 240.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | -1.757 ms / 0.683x |
| Runtime 2 (8 Workers) | 0 | 20.400 ms | 21.000 ms | 20.400 ms | 21.000 ms | 0.000 ms | 0.0 / 0.0 / 240.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -16.614 ms / 0.186x |
| Runtime 2 (1 Worker) | 0 | 23.757 ms | 23.900 ms | 23.800 ms | 24.000 ms | 0.043 ms | 0.0 / 0.0 / 240.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -19.971 ms / 0.159x |
| Runtime 2 (2 Workers) | 0 | 24.129 ms | 22.500 ms | 24.200 ms | 22.500 ms | 0.071 ms | 0.0 / 0.0 / 240.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -20.343 ms / 0.157x |
| Runtime 2 (4 Workers) | 0 | 24.829 ms | 24.400 ms | 24.871 ms | 24.500 ms | 0.043 ms | 0.0 / 0.0 / 240.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -21.043 ms / 0.152x |

### Core Refresh

- Category: `Refresh`
- Requested work: Refresh the current core-list view without changing row count.
- Correctness check: 40 .benchmark-core-item nodes must remain rendered and the active core-list refresh token must change after the refresh action.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 1.700 ms | 1.700 ms | 16.857 ms | 16.700 ms | 15.157 ms | 0.0 / 1.0 / 0.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 2 (1 Worker) | 0 | 2.529 ms | 2.100 ms | 16.643 ms | 16.300 ms | 14.114 ms | 0.0 / 1.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -0.829 ms / 0.672x |
| Runtime 2 (4 Workers) | 0 | 2.986 ms | 3.200 ms | 16.271 ms | 16.200 ms | 13.286 ms | 0.0 / 1.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -1.286 ms / 0.569x |
| Runtime 2 (8 Workers) | 0 | 3.386 ms | 3.800 ms | 16.314 ms | 16.200 ms | 12.929 ms | 0.0 / 1.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -1.686 ms / 0.502x |
| Runtime 2 (2 Workers) | 0 | 3.571 ms | 3.500 ms | 16.229 ms | 16.000 ms | 12.657 ms | 0.0 / 1.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -1.871 ms / 0.476x |
| Runtime 1 | 0 | 3.857 ms | 2.600 ms | 16.457 ms | 16.500 ms | 12.600 ms | 0.0 / 1.0 / 0.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | -2.157 ms / 0.441x |

### Core Append

- Category: `Structural Churn`
- Requested work: Append 100 rows after a 240-row stress core list.
- Correctness check: 340 .benchmark-core-item nodes must exist and the last row-id must become 1100.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 3.829 ms | 3.300 ms | 21.071 ms | 20.300 ms | 17.243 ms | 100.0 / 0.0 / 0.0 | 100.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 1 | 0 | 10.257 ms | 9.900 ms | 10.986 ms | 10.800 ms | 0.729 ms | 1.0 / 0.0 / 0.0 | 100.0 / 0.0 | n/a | 0.0 / 0.000 ms | -6.428 ms / 0.373x |
| Runtime 2 (1 Worker) | 0 | 36.843 ms | 36.900 ms | 37.471 ms | 37.000 ms | 0.629 ms | 1.0 / 0.0 / 0.0 | 2.0 / 0.0 | 1.0 / 21.657 ms / 340.0 | 0.0 / 0.000 ms | -33.014 ms / 0.104x |
| Runtime 2 (8 Workers) | 0 | 46.243 ms | 46.100 ms | 46.286 ms | 46.100 ms | 0.043 ms | 4.0 / 0.0 / 0.0 | 102.0 / 0.0 | 1.0 / 22.457 ms / 340.0 | 0.0 / 0.000 ms | -42.414 ms / 0.083x |
| Runtime 2 (4 Workers) | 0 | 47.800 ms | 51.000 ms | 47.900 ms | 51.200 ms | 0.100 ms | 4.0 / 0.0 / 0.0 | 102.0 / 0.0 | 1.0 / 23.514 ms / 340.0 | 0.0 / 0.000 ms | -43.971 ms / 0.080x |
| Runtime 2 (2 Workers) | 0 | 53.943 ms | 55.500 ms | 57.129 ms | 58.000 ms | 3.186 ms | 4.0 / 0.0 / 0.0 | 102.0 / 0.0 | 1.0 / 26.900 ms / 340.0 | 0.0 / 0.000 ms | -50.114 ms / 0.071x |

### Core Filter

- Category: `Structural Churn`
- Requested work: Filter the 240-row stress core list down to rows whose stable ID is divisible by 3.
- Correctness check: 80 .benchmark-core-item nodes must remain and every row-id must be divisible by 3.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 2.129 ms | 2.100 ms | 19.186 ms | 18.900 ms | 17.057 ms | 160.0 / 0.0 / 0.0 | 0.0 / 160.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 1 | 0 | 4.029 ms | 3.300 ms | 9.629 ms | 9.400 ms | 5.600 ms | 160.0 / 0.0 / 0.0 | 0.0 / 160.0 | n/a | 0.0 / 0.000 ms | -1.900 ms / 0.528x |
| Runtime 2 (4 Workers) | 0 | 16.029 ms | 17.400 ms | 17.814 ms | 17.400 ms | 1.786 ms | 64.0 / 0.0 / 0.0 | 50.0 / 63.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -13.900 ms / 0.133x |
| Runtime 2 (8 Workers) | 0 | 17.700 ms | 19.600 ms | 19.671 ms | 19.900 ms | 1.971 ms | 64.0 / 0.0 / 0.0 | 50.0 / 63.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -15.571 ms / 0.120x |
| Runtime 2 (2 Workers) | 0 | 19.629 ms | 20.500 ms | 20.557 ms | 20.500 ms | 0.929 ms | 45.0 / 0.0 / 0.0 | 60.0 / 44.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -17.500 ms / 0.108x |
| Runtime 2 (1 Worker) | 0 | 20.600 ms | 22.000 ms | 22.614 ms | 22.000 ms | 2.014 ms | 82.0 / 0.0 / 0.0 | 40.0 / 81.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -18.471 ms / 0.103x |

### Content Render

- Category: `Initial Render`
- Requested work: Render 12 nested content cards from empty state.
- Correctness check: 12 .benchmark-content-card nodes must exist.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 2.700 ms | 2.400 ms | 21.957 ms | 22.200 ms | 19.257 ms | 12.0 / 0.0 / 0.0 | 12.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 1 | 0 | 5.986 ms | 5.100 ms | 10.114 ms | 10.400 ms | 4.129 ms | 1.0 / 0.0 / 0.0 | 12.0 / 0.0 | n/a | 0.0 / 0.000 ms | -3.286 ms / 0.451x |
| Runtime 2 (2 Workers) | 0 | 11.229 ms | 9.500 ms | 17.014 ms | 16.500 ms | 5.786 ms | 3.0 / 0.0 / 0.0 | 14.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -8.529 ms / 0.240x |
| Runtime 2 (8 Workers) | 0 | 12.200 ms | 12.700 ms | 16.743 ms | 16.600 ms | 4.543 ms | 3.0 / 0.0 / 0.0 | 14.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -9.500 ms / 0.221x |
| Runtime 2 (1 Worker) | 0 | 12.329 ms | 14.100 ms | 13.657 ms | 14.100 ms | 1.329 ms | 3.0 / 0.0 / 0.0 | 14.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -9.629 ms / 0.219x |
| Runtime 2 (4 Workers) | 0 | 13.343 ms | 12.100 ms | 17.771 ms | 16.100 ms | 4.429 ms | 3.0 / 0.0 / 0.0 | 14.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -10.643 ms / 0.202x |

### Content Update

- Category: `Targeted Update`
- Requested work: Update the text and status fields of the 12 rendered content cards.
- Correctness check: All .benchmark-content-status values must become 'live' and titles must include '(Updated)'.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 2.271 ms | 2.000 ms | 19.214 ms | 19.400 ms | 16.943 ms | 0.0 / 0.0 / 48.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 1 | 0 | 3.314 ms | 3.100 ms | 16.686 ms | 16.700 ms | 13.371 ms | 0.0 / 0.0 / 48.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | -1.043 ms / 0.685x |
| Runtime 2 (1 Worker) | 0 | 8.357 ms | 7.000 ms | 16.771 ms | 17.100 ms | 8.414 ms | 0.0 / 0.0 / 48.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -6.086 ms / 0.272x |
| Runtime 2 (8 Workers) | 0 | 8.771 ms | 8.200 ms | 16.729 ms | 16.000 ms | 7.957 ms | 0.0 / 0.0 / 48.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -6.500 ms / 0.259x |
| Runtime 2 (2 Workers) | 0 | 9.171 ms | 6.100 ms | 17.414 ms | 16.800 ms | 8.243 ms | 0.0 / 0.0 / 48.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -6.900 ms / 0.248x |
| Runtime 2 (4 Workers) | 0 | 9.629 ms | 9.100 ms | 16.386 ms | 16.400 ms | 6.757 ms | 0.0 / 0.0 / 48.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -7.358 ms / 0.236x |

### Content Refresh

- Category: `Refresh`
- Requested work: Refresh the current content-card view without changing card count.
- Correctness check: 12 .benchmark-content-card nodes must remain rendered and the active content refresh token must change after the refresh action.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 1.614 ms | 1.600 ms | 16.529 ms | 16.400 ms | 14.914 ms | 0.0 / 13.0 / 0.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 1 | 0 | 4.700 ms | 5.600 ms | 16.300 ms | 16.400 ms | 11.600 ms | 0.0 / 13.0 / 0.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | -3.086 ms / 0.343x |
| Runtime 2 (1 Worker) | 0 | 7.329 ms | 9.100 ms | 16.929 ms | 16.700 ms | 9.600 ms | 0.0 / 13.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -5.715 ms / 0.220x |
| Runtime 2 (4 Workers) | 0 | 8.386 ms | 10.200 ms | 16.486 ms | 16.400 ms | 8.100 ms | 0.0 / 13.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -6.772 ms / 0.192x |
| Runtime 2 (2 Workers) | 0 | 8.829 ms | 9.000 ms | 16.400 ms | 16.400 ms | 7.571 ms | 0.0 / 13.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -7.215 ms / 0.183x |
| Runtime 2 (8 Workers) | 0 | 9.157 ms | 6.900 ms | 17.057 ms | 16.100 ms | 7.900 ms | 0.0 / 13.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -7.543 ms / 0.176x |

### Primitive Render

- Category: `Primitive Render`
- Requested work: Render 200 flat primitive host nodes from empty state.
- Correctness check: 200 .benchmark-primitive-row nodes must exist after the render action.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 3.000 ms | 2.800 ms | 20.829 ms | 20.000 ms | 17.829 ms | 200.0 / 0.0 / 0.0 | 200.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 2 (8 Workers) | 0 | 5.214 ms | 4.900 ms | 16.500 ms | 16.700 ms | 11.286 ms | 1.0 / 0.0 / 0.0 | 200.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -2.214 ms / 0.575x |
| Runtime 2 (1 Worker) | 0 | 5.257 ms | 5.200 ms | 15.157 ms | 15.000 ms | 9.900 ms | 1.0 / 0.0 / 0.0 | 200.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -2.257 ms / 0.571x |
| Runtime 1 | 0 | 6.100 ms | 5.300 ms | 12.129 ms | 12.400 ms | 6.029 ms | 1.0 / 0.0 / 0.0 | 200.0 / 0.0 | n/a | 0.0 / 0.000 ms | -3.100 ms / 0.492x |
| Runtime 2 (4 Workers) | 0 | 7.257 ms | 6.000 ms | 16.686 ms | 17.100 ms | 9.429 ms | 1.0 / 0.0 / 0.0 | 200.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -4.257 ms / 0.413x |
| Runtime 2 (2 Workers) | 0 | 8.043 ms | 6.200 ms | 16.414 ms | 16.400 ms | 8.371 ms | 1.0 / 0.0 / 0.0 | 200.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -5.043 ms / 0.373x |

### Primitive Text Update

- Category: `Primitive Update`
- Requested work: Update only the text content of the existing 200 primitive host nodes.
- Correctness check: All .benchmark-primitive-label nodes must include '(Live)' while row count stays at 200.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 3.000 ms | 2.900 ms | 16.486 ms | 16.600 ms | 13.486 ms | 0.0 / 0.0 / 200.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 1 | 0 | 4.057 ms | 4.200 ms | 16.700 ms | 16.400 ms | 12.643 ms | 0.0 / 0.0 / 200.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | -1.057 ms / 0.739x |
| Runtime 2 (1 Worker) | 0 | 4.086 ms | 3.500 ms | 16.700 ms | 16.500 ms | 12.614 ms | 0.0 / 0.0 / 200.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -1.086 ms / 0.734x |
| Runtime 2 (2 Workers) | 0 | 5.414 ms | 4.700 ms | 16.443 ms | 16.100 ms | 11.029 ms | 0.0 / 0.0 / 200.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -2.414 ms / 0.554x |
| Runtime 2 (4 Workers) | 0 | 5.800 ms | 6.800 ms | 16.443 ms | 16.100 ms | 10.643 ms | 0.0 / 0.0 / 200.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -2.800 ms / 0.517x |
| Runtime 2 (8 Workers) | 0 | 6.171 ms | 6.700 ms | 16.543 ms | 16.600 ms | 10.371 ms | 0.0 / 0.0 / 200.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -3.171 ms / 0.486x |

### Primitive Attribute Update

- Category: `Primitive Update`
- Requested work: Update only the attributes and classes of the existing 200 primitive host nodes.
- Correctness check: All .benchmark-primitive-row nodes must keep count 200 and switch to data-primitive-state='active'.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 2.086 ms | 2.000 ms | 19.329 ms | 19.300 ms | 17.243 ms | 0.0 / 400.0 / 0.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 2 (4 Workers) | 0 | 4.929 ms | 5.000 ms | 16.371 ms | 16.200 ms | 11.443 ms | 0.0 / 400.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -2.843 ms / 0.423x |
| Runtime 1 | 0 | 5.314 ms | 5.200 ms | 16.800 ms | 17.000 ms | 11.486 ms | 0.0 / 400.0 / 0.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | -3.228 ms / 0.393x |
| Runtime 2 (8 Workers) | 0 | 6.457 ms | 5.300 ms | 16.557 ms | 16.400 ms | 10.100 ms | 0.0 / 400.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -4.371 ms / 0.323x |
| Runtime 2 (2 Workers) | 0 | 6.614 ms | 5.800 ms | 16.329 ms | 16.000 ms | 9.714 ms | 0.0 / 400.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -4.528 ms / 0.315x |
| Runtime 2 (1 Worker) | 0 | 7.143 ms | 6.100 ms | 16.600 ms | 16.700 ms | 9.457 ms | 0.0 / 400.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -5.057 ms / 0.292x |

### Primitive Append

- Category: `Primitive Churn`
- Requested work: Append 100 primitive host nodes after the initial 200-row primitive grid.
- Correctness check: 300 .benchmark-primitive-row nodes must exist and the last data-primitive-id must become 300.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 2.486 ms | 2.600 ms | 19.443 ms | 19.400 ms | 16.957 ms | 100.0 / 1.0 / 0.0 | 100.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 2 (1 Worker) | 0 | 4.971 ms | 4.900 ms | 15.143 ms | 14.800 ms | 10.171 ms | 1.0 / 1.0 / 0.0 | 100.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -2.485 ms / 0.500x |
| Runtime 2 (4 Workers) | 0 | 5.357 ms | 4.600 ms | 16.471 ms | 16.400 ms | 11.114 ms | 1.0 / 1.0 / 0.0 | 100.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -2.871 ms / 0.464x |
| Runtime 2 (8 Workers) | 0 | 5.400 ms | 4.900 ms | 15.986 ms | 15.900 ms | 10.586 ms | 1.0 / 1.0 / 0.0 | 100.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -2.914 ms / 0.460x |
| Runtime 1 | 0 | 5.943 ms | 5.200 ms | 11.600 ms | 11.300 ms | 5.657 ms | 1.0 / 1.0 / 0.0 | 100.0 / 0.0 | n/a | 0.0 / 0.000 ms | -3.457 ms / 0.418x |
| Runtime 2 (2 Workers) | 0 | 7.400 ms | 4.800 ms | 16.700 ms | 16.500 ms | 9.300 ms | 1.0 / 1.0 / 0.0 | 100.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -4.914 ms / 0.336x |

### Primitive Remove

- Category: `Primitive Churn`
- Requested work: Remove 100 primitive host nodes from the trailing edge of the 200-row primitive grid.
- Correctness check: 100 .benchmark-primitive-row nodes must remain and the last data-primitive-id must become 100.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 2.857 ms | 2.700 ms | 20.400 ms | 19.700 ms | 17.543 ms | 100.0 / 1.0 / 0.0 | 0.0 / 100.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 2 (1 Worker) | 0 | 3.643 ms | 3.500 ms | 13.000 ms | 13.100 ms | 9.357 ms | 100.0 / 1.0 / 0.0 | 0.0 / 100.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -0.786 ms / 0.784x |
| Runtime 2 (8 Workers) | 0 | 3.914 ms | 4.000 ms | 16.443 ms | 16.500 ms | 12.529 ms | 100.0 / 1.0 / 0.0 | 0.0 / 100.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -1.057 ms / 0.730x |
| Runtime 2 (4 Workers) | 0 | 5.300 ms | 5.200 ms | 16.371 ms | 16.400 ms | 11.071 ms | 100.0 / 1.0 / 0.0 | 0.0 / 100.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -2.443 ms / 0.539x |
| Runtime 2 (2 Workers) | 0 | 5.400 ms | 4.800 ms | 16.014 ms | 16.300 ms | 10.614 ms | 100.0 / 1.0 / 0.0 | 0.0 / 100.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -2.543 ms / 0.529x |
| Runtime 1 | 0 | 13.786 ms | 4.900 ms | 18.071 ms | 11.300 ms | 4.286 ms | 100.0 / 1.0 / 0.0 | 0.0 / 100.0 | n/a | 0.0 / 0.000 ms | -10.929 ms / 0.207x |

### Deep Tree Render

- Category: `Initial Render`
- Requested work: Render the recursive deep-tree benchmark view until the benchmark leaf becomes visible.
- Correctness check: #benchmark-deep-leaf must exist after the render action.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 2.971 ms | 3.000 ms | 14.329 ms | 15.900 ms | 11.357 ms | 1.0 / 3.0 / 0.0 | 1.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 1 | 0 | 3.471 ms | 3.500 ms | 14.029 ms | 15.200 ms | 10.557 ms | 2.0 / 3.0 / 0.0 | 3.0 / 0.0 | n/a | 0.0 / 0.000 ms | -0.500 ms / 0.856x |
| Runtime 2 (4 Workers) | 0 | 6.300 ms | 6.300 ms | 17.071 ms | 17.100 ms | 10.771 ms | 3.0 / 3.0 / 0.0 | 4.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -3.329 ms / 0.472x |
| Runtime 2 (1 Worker) | 0 | 8.529 ms | 7.300 ms | 16.729 ms | 17.000 ms | 8.200 ms | 3.0 / 3.0 / 0.0 | 4.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -5.558 ms / 0.348x |
| Runtime 2 (8 Workers) | 0 | 9.514 ms | 9.900 ms | 16.786 ms | 17.000 ms | 7.271 ms | 3.0 / 3.0 / 0.0 | 4.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -6.543 ms / 0.312x |
| Runtime 2 (2 Workers) | 0 | 9.557 ms | 8.800 ms | 17.014 ms | 16.500 ms | 7.457 ms | 3.0 / 3.0 / 0.0 | 4.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -6.586 ms / 0.311x |

### Deep Tree Update

- Category: `Targeted Update`
- Requested work: Update the 60-level compliance tree in place so every nested level carries the new revision marker.
- Correctness check: The deep-tree root and every nested level must keep the same shape while the deep-tree revision marker changes.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 3.257 ms | 3.600 ms | 16.143 ms | 16.200 ms | 12.886 ms | 0.0 / 4.0 / 61.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 1 | 0 | 3.629 ms | 4.200 ms | 17.043 ms | 17.000 ms | 13.414 ms | 0.0 / 4.0 / 61.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | -0.372 ms / 0.897x |
| Runtime 2 (2 Workers) | 0 | 4.929 ms | 3.600 ms | 16.657 ms | 16.900 ms | 11.729 ms | 0.0 / 4.0 / 61.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -1.672 ms / 0.661x |
| Runtime 2 (1 Worker) | 0 | 7.586 ms | 5.700 ms | 16.814 ms | 16.700 ms | 9.229 ms | 0.0 / 4.0 / 61.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -4.329 ms / 0.429x |
| Runtime 2 (8 Workers) | 0 | 10.743 ms | 9.000 ms | 16.786 ms | 16.600 ms | 6.043 ms | 0.0 / 4.0 / 61.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -7.486 ms / 0.303x |
| Runtime 2 (4 Workers) | 0 | 11.229 ms | 9.000 ms | 17.414 ms | 16.800 ms | 6.186 ms | 0.0 / 4.0 / 61.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -7.972 ms / 0.290x |

### Deep Tree Refresh

- Category: `Refresh`
- Requested work: Refresh the current deep-tree view without changing depth while the deep-tree refresh marker changes.
- Correctness check: The deep-tree root and leaf must remain rendered and the deep-tree refresh token must change after refresh.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 1.843 ms | 2.100 ms | 16.386 ms | 16.500 ms | 14.543 ms | 0.0 / 2.0 / 0.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 1 | 0 | 3.129 ms | 2.300 ms | 16.243 ms | 16.100 ms | 13.114 ms | 0.0 / 2.0 / 0.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | -1.286 ms / 0.589x |
| Runtime 2 (2 Workers) | 0 | 5.600 ms | 3.700 ms | 16.257 ms | 16.100 ms | 10.657 ms | 0.0 / 2.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -3.757 ms / 0.329x |
| Runtime 2 (4 Workers) | 0 | 6.500 ms | 7.000 ms | 16.029 ms | 15.800 ms | 9.529 ms | 0.0 / 2.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -4.657 ms / 0.284x |
| Runtime 2 (8 Workers) | 0 | 6.600 ms | 6.100 ms | 16.314 ms | 16.200 ms | 9.714 ms | 0.0 / 2.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -4.757 ms / 0.279x |
| Runtime 2 (1 Worker) | 0 | 7.343 ms | 6.500 ms | 16.271 ms | 16.100 ms | 8.929 ms | 0.0 / 2.0 / 0.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -5.500 ms / 0.251x |

### Enterprise Subtree Update

- Category: `Targeted Update`
- Requested work: Update one nested enterprise workspace section in place while preserving sibling sections and record count.
- Correctness check: 6 enterprise sections and 30 records must remain rendered while the target section revision and statuses change in place.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 2.471 ms | 2.200 ms | 19.757 ms | 19.300 ms | 17.286 ms | 0.0 / 13.0 / 38.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 2 (2 Workers) | 0 | 4.971 ms | 4.100 ms | 16.914 ms | 17.000 ms | 11.943 ms | 0.0 / 13.0 / 38.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -2.500 ms / 0.497x |
| Runtime 1 | 0 | 5.743 ms | 4.200 ms | 16.714 ms | 16.700 ms | 10.971 ms | 0.0 / 13.0 / 38.0 | 0.0 / 0.0 | n/a | 0.0 / 0.000 ms | -3.272 ms / 0.430x |
| Runtime 2 (1 Worker) | 0 | 5.900 ms | 4.500 ms | 16.514 ms | 16.300 ms | 10.614 ms | 0.0 / 13.0 / 38.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -3.429 ms / 0.419x |
| Runtime 2 (4 Workers) | 0 | 6.014 ms | 4.900 ms | 16.271 ms | 16.100 ms | 10.257 ms | 0.0 / 13.0 / 38.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -3.543 ms / 0.411x |
| Runtime 2 (8 Workers) | 0 | 7.014 ms | 4.500 ms | 16.943 ms | 16.900 ms | 9.929 ms | 0.0 / 13.0 / 38.0 | 0.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -4.543 ms / 0.352x |

### Hook Grid Render

- Category: `Initial Render`
- Requested work: Render the 40-cell hook-heavy grid view.
- Correctness check: Exactly 40 .benchmark-hook-node elements must exist after the render action.
- Finish line: DOM-ready plus next requestAnimationFrame paint proxy.

| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| React 19.2.4 | 0 | 2.471 ms | 2.100 ms | 22.100 ms | 19.300 ms | 19.629 ms | 40.0 / 2.0 / 0.0 | 40.0 / 0.0 | n/a | 0.0 / 0.000 ms | +0.000 ms / 1.000x |
| Runtime 1 | 0 | 6.686 ms | 5.000 ms | 16.414 ms | 16.500 ms | 9.729 ms | 1.0 / 2.0 / 0.0 | 40.0 / 0.0 | n/a | 0.0 / 0.000 ms | -4.215 ms / 0.370x |
| Runtime 2 (1 Worker) | 0 | 16.029 ms | 18.900 ms | 18.343 ms | 19.000 ms | 2.314 ms | 1.0 / 2.0 / 0.0 | 40.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -13.558 ms / 0.154x |
| Runtime 2 (2 Workers) | 0 | 20.486 ms | 19.100 ms | 22.429 ms | 19.200 ms | 1.943 ms | 1.0 / 2.0 / 0.0 | 40.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -18.015 ms / 0.121x |
| Runtime 2 (8 Workers) | 0 | 20.571 ms | 20.100 ms | 22.329 ms | 20.100 ms | 1.757 ms | 1.0 / 2.0 / 0.0 | 40.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -18.100 ms / 0.120x |
| Runtime 2 (4 Workers) | 0 | 21.271 ms | 21.800 ms | 21.300 ms | 21.800 ms | 0.029 ms | 1.0 / 2.0 / 0.0 | 40.0 / 0.0 | 0.0 / 0.000 ms / 0.0 | 0.0 / 0.000 ms | -18.800 ms / 0.116x |
