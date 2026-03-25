# Example 100 Benchmarks

This example includes two benchmark styles:

- microbenchmarks for store and synthetic `Send` hot paths
- an SLA sweep that increases concurrent synthetic clients until the server breaches a latency target

The benchmark code lives in:

- `examples/100-ai-chat-wizard/server/app/benchmark_test.go`
- `examples/100-ai-chat-wizard/server/app/benchmark_sla_test.go`

## What Is Being Measured

The SLA sweep is measuring **max total concurrent synthetic clients under a p95 latency target** for the server-side `Send` path.

It does this by:

1. pinning `GOMAXPROCS` to a requested core count
2. starting at `1` concurrent synthetic client
3. increasing clients one step at a time
4. measuring request latencies for that burst
5. stopping when `p95` exceeds the SLA

This is a **server-side synthetic benchmark**:

- it exercises the example's gRPC service code path and SQLite persistence
- it uses the synthetic benchmark provider instead of real upstream model latency
- it does not include browser/WASM rendering, real network jitter, or provider rate limits

## Commands

Microbenchmarks:

```powershell
go test ./examples/100-ai-chat-wizard/server/app -run '^$' -bench 'Benchmark(StoreCorePaths|SendClientsPerCore)' -benchmem
```

SLA sweep:

```powershell
$env:CHAT_WIZARD_BENCH_SLA_MS = "100"
go test ./examples/100-ai-chat-wizard/server/app -run TestSendSLASweep -v
```

Useful knobs:

- `CHAT_WIZARD_BENCH_MAX_CORES`
- `CHAT_WIZARD_BENCH_MAX_CLIENTS`
- `CHAT_WIZARD_BENCH_BURST_RUNS`
- `CHAT_WIZARD_BENCH_PREDICT_CORES`

Example research run:

```powershell
$env:CHAT_WIZARD_BENCH_SLA_MS='100'
$env:CHAT_WIZARD_BENCH_MAX_CORES='8'
$env:CHAT_WIZARD_BENCH_MAX_CLIENTS='512'
$env:CHAT_WIZARD_BENCH_BURST_RUNS='1'
$env:CHAT_WIZARD_BENCH_PREDICT_CORES='32'
go test ./examples/100-ai-chat-wizard/server/app -run TestSendSLASweep -v
```

## How To Read The Output

The sweep emits four kinds of lines.

### `core summary`

Example:

```text
core summary: cores=4 max_clients_under_sla=87 sla=100ms
```

Meaning:

- with `GOMAXPROCS=4`
- the sweep found that `87` concurrent synthetic clients stayed under the SLA
- the next client count breached the SLA

This is the direct measured point for that core count.

### `curve point`

Example:

```text
curve point: cores=4 total_clients=87 clients_per_core=21.75 scaling_vs_1_core=0.392 clipped_by_cap=false
```

Meaning:

- `total_clients` is the raw measured total for that core count
- `clients_per_core` is `total_clients / cores`
- `scaling_vs_1_core` is `total_clients_at_n / total_clients_at_1`
- `clipped_by_cap=true` means the test hit `CHAT_WIZARD_BENCH_MAX_CLIENTS` before it found the real knee

If the first point is clipped, the baseline is incomplete and projections will be poor. Raise the client cap and rerun.

### `rollup point`

Example:

```text
rollup point: cores=4 raw_total_clients=87 cumulative_total_clients=542 cumulative_clients_per_core=135.50 clipped_by_cap=false
```

Meaning:

- `raw_total_clients` is the measured total for exactly that core count
- `cumulative_total_clients` is the running sum from `1..n` cores
- `cumulative_clients_per_core` is `cumulative_total_clients / cores`

This rollup is a planning model. It treats each additional core tier as additive capacity on top of the prior tiers. That is why `32-core` and `64-core` predictions are built from the cumulative curve rather than the noisy raw curve.

### `projection`

Example:

```text
projection: method=linear_regression_rollup cores=32 predicted_total_clients=3430.19 slope=102.9524 intercept=135.7143 source_points=8
```

Meaning:

- a linear regression was fit over the uncensored cumulative rollup points
- the prediction target was `32` cores
- the output is a **projection**, not a measured result

## Latest Measured Research Data

Latest `100 ms p95` sweep on this workstation with `1..8` cores and client cap `512`:

Raw measured totals:

- `1 core: 222 clients`
- `2 cores: 127 clients`
- `3 cores: 106 clients`
- `4 cores: 87 clients`
- `5 cores: 118 clients`
- `6 cores: 102 clients`
- `7 cores: 91 clients`
- `8 cores: 96 clients`

Derived raw metrics:

- `1 core`: `222.00 clients/core`, scaling `1.000x`
- `2 cores`: `63.50 clients/core`, scaling `0.572x`
- `3 cores`: `35.33 clients/core`, scaling `0.477x`
- `4 cores`: `21.75 clients/core`, scaling `0.392x`
- `5 cores`: `23.60 clients/core`, scaling `0.532x`
- `6 cores`: `17.00 clients/core`, scaling `0.459x`
- `7 cores`: `13.00 clients/core`, scaling `0.410x`
- `8 cores`: `12.00 clients/core`, scaling `0.432x`

Cumulative rollup:

- `1 core: 222`
- `2 cores: 349`
- `3 cores: 455`
- `4 cores: 542`
- `5 cores: 660`
- `6 cores: 762`
- `7 cores: 853`
- `8 cores: 949`

Projected totals from the cumulative rollup regression:

- `32 cores: ~3430 clients`
- `64 cores: ~6725 clients`

## What The Data Is Saying

The raw measured per-core curve is not scaling cleanly upward. On this synthetic server path, the per-core efficiency drops sharply after the first core. That usually means a shared bottleneck is dominating, such as:

- SQLite write contention
- shared mutex/coordination in the request path
- synthetic burst behavior that front-loads contention rather than smoothing it out

The cumulative rollup projection is therefore a **capacity-planning estimate**, not proof that one 32-core box will actually sustain the predicted total.

Use the results like this:

- use `core summary` for direct measured points
- use `curve point` to reason about efficiency and scaling quality
- use `rollup point` and `projection` as planning estimates

Do not treat the projection as production truth until it is confirmed with:

- a higher-fidelity bridge/network load test
- a real provider latency model or a stub that matches it
- production-like persistence settings
