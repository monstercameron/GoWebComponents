#!/usr/bin/env python3
"""Compare paired Example 201 browser benchmark captures.

The reports are produced by the Playwright harness with the same seed and
scenario set.  DOM Ready representative timing is the primary metric used by
the report itself.  The gate keeps the plan's existing +/-30% browser noise
floor explicit and requires the candidate geometric mean to be no slower than
the historical baseline.  A failed gate is useful evidence; this script never
normalizes away a real regression.
"""

from __future__ import annotations

import argparse
import json
import math
import statistics
import sys
from pathlib import Path


def load_runtime_report(path: Path) -> tuple[dict, dict, dict[str, dict]]:
    payload = json.loads(path.read_text(encoding="utf-8"))
    report = payload.get("getReport")
    if not isinstance(report, dict):
        raise ValueError(f"{path}: missing getReport")
    frameworks = report.get("getFrameworks")
    if not isinstance(frameworks, list):
        raise ValueError(f"{path}: missing getFrameworks")
    runtime = [item for item in frameworks if item.get("getFramework") == "runtime1"]
    if len(runtime) != 1:
        raise ValueError(f"{path}: expected one runtime1 framework, found {len(runtime)}")
    scenarios = runtime[0].get("getScenarioResults")
    if not isinstance(scenarios, list) or not scenarios:
        raise ValueError(f"{path}: runtime1 has no scenario results")
    result: dict[str, dict] = {}
    for item in scenarios:
        scenario_id = item.get("getScenarioID")
        if not isinstance(scenario_id, str) or not scenario_id:
            raise ValueError(f"{path}: scenario has no getScenarioID")
        if scenario_id in result:
            raise ValueError(f"{path}: duplicate runtime1 scenario {scenario_id}")
        samples = item.get("getDomReadySamplesMs")
        if not isinstance(samples, list) or not samples:
            raise ValueError(f"{path}: {scenario_id} has no raw timing samples")
        if any(not isinstance(value, (int, float)) or not math.isfinite(value) or value <= 0 for value in samples):
            raise ValueError(f"{path}: {scenario_id} has invalid raw timing samples {samples!r}")
        result[scenario_id] = item
    return payload, report, result


def geometric_mean(values: list[float]) -> float:
    return math.exp(sum(math.log(value) for value in values) / len(values))


def capture_medians(result: dict[str, dict]) -> dict[str, float]:
    return {scenario_id: statistics.median(item["getDomReadySamplesMs"])
            for scenario_id, item in result.items()}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--baseline", type=Path, action="append", required=True, help="one v6 report; repeat for captures")
    parser.add_argument("--candidate", type=Path, action="append", required=True, help="one v7 report; repeat for captures")
    parser.add_argument("--json", type=Path, required=True)
    parser.add_argument("--markdown", type=Path, required=True)
    parser.add_argument("--max-regress-pct", type=float, default=30.0)
    args = parser.parse_args()

    try:
        if len(args.baseline) != len(args.candidate) or len(args.baseline) < 3:
            raise ValueError("provide at least three paired baseline/candidate reports")
        baselines = [load_runtime_report(path) for path in args.baseline]
        candidates = [load_runtime_report(path) for path in args.candidate]
        baseline_payload, baseline_report, baseline = baselines[0]
        candidate_payload, candidate_report, candidate = candidates[0]
        for (_, report, result), (_, candidate_capture_report, candidate_capture) in zip(baselines, candidates):
            if baseline_payload.get("getBrowser") != candidate_payload.get("getBrowser"):
                raise ValueError("baseline and candidate use different browsers")
            if report.get("getScenarioOrder") != candidate_capture_report.get("getScenarioOrder"):
                raise ValueError("baseline and candidate scenario orders differ")
            if report.get("getSeed") != candidate_capture_report.get("getSeed"):
                raise ValueError("baseline and candidate seeds differ")
            if set(result) != set(candidate_capture):
                raise ValueError("baseline and candidate scenario sets differ")
        if len(baseline) != 19:
            raise ValueError(f"expected all 19 Example 201 scenarios, found {len(baseline)}")
        if set(baseline_report.get("getScenarioOrder", [])) != set(baseline) or len(baseline_report["getScenarioOrder"]) != 19:
            raise ValueError("scenario order must contain all 19 scenarios exactly once")
        if any(payload.get("getBrowser") != baseline_payload.get("getBrowser") or
               report.get("getScenarioOrder") != baseline_report.get("getScenarioOrder") or
               report.get("getSeed") != baseline_report.get("getSeed") or
               set(result) != set(baseline)
               for payload, report, result in baselines[1:] + candidates[1:]):
            raise ValueError("captures do not share browser, seed, order, or scenario set")
        if not math.isfinite(args.max_regress_pct) or args.max_regress_pct < 0:
            raise ValueError("--max-regress-pct must be finite and non-negative")
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"Example 201 comparison error: {exc}", file=sys.stderr)
        return 2

    rows = []
    baseline_captures = [capture_medians(result) for _, _, result in baselines]
    candidate_captures = [capture_medians(result) for _, _, result in candidates]
    for scenario_id in baseline_report["getScenarioOrder"]:
        if scenario_id not in baseline:
            continue
        baseline_values = [capture[scenario_id] for capture in baseline_captures]
        candidate_values = [capture[scenario_id] for capture in candidate_captures]
        baseline_ms = float(statistics.median(baseline_values))
        candidate_ms = float(statistics.median(candidate_values))
        delta_pct = (candidate_ms / baseline_ms - 1.0) * 100.0
        rows.append({
            "scenarioID": scenario_id,
            "baselineMs": baseline_ms,
            "candidateMs": candidate_ms,
            "baselineCaptureMediansMs": baseline_values,
            "candidateCaptureMediansMs": candidate_values,
            "baselineSpreadPct": (max(baseline_values) / min(baseline_values) - 1.0) * 100.0,
            "candidateSpreadPct": (max(candidate_values) / min(candidate_values) - 1.0) * 100.0,
            "deltaPct": delta_pct,
            "regressedBeyondNoise": delta_pct > args.max_regress_pct,
        })
    baseline_mean = geometric_mean([row["baselineMs"] for row in rows])
    candidate_mean = geometric_mean([row["candidateMs"] for row in rows])
    geomean_delta_pct = (candidate_mean / baseline_mean - 1.0) * 100.0
    failures = [row["scenarioID"] for row in rows if row["regressedBeyondNoise"]]
    gate_passed = not failures and geomean_delta_pct <= 0.0
    result = {
        "schema": "example201-v6-v7/v2",
        "captureCount": len(args.baseline),
        "baseline": {"paths": [str(path) for path in args.baseline]},
        "candidate": {"paths": [str(path) for path in args.candidate]},
        "browser": candidate_payload.get("getBrowser"),
        "seed": candidate_report.get("getSeed"),
        "scenarioCount": len(rows),
        "maxRegressPct": args.max_regress_pct,
        "baselineGeomeanMs": baseline_mean,
        "candidateGeomeanMs": candidate_mean,
        "geomeanDeltaPct": geomean_delta_pct,
        "regressedBeyondNoise": failures,
        "gatePassed": gate_passed,
        "scenarios": rows,
    }
    args.json.parent.mkdir(parents=True, exist_ok=True)
    args.json.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    lines = [
        "# Example 201 v6/v7 Browser Gate",
        "",
        f"- Browser: `{result['browser']}`",
        f"- Seed: `{result['seed']}`",
        f"- Paired captures: `{result['captureCount']}` (median of raw samples, then median across captures)",
        f"- Scenarios: `{result['scenarioCount']}`",
        f"- Noise floor: `+{args.max_regress_pct:.1f}%` per scenario",
        f"- Baseline geomean: `{baseline_mean:.3f} ms`",
        f"- Candidate geomean: `{candidate_mean:.3f} ms`",
        f"- Geomean delta: `{geomean_delta_pct:+.2f}%`",
        f"- Gate: **{'PASS' if gate_passed else 'FAIL'}**",
        "",
        "| Scenario | v6 median (ms) | v7 median (ms) | Delta | v6 spread | v7 spread | Beyond noise |",
        "| --- | ---: | ---: | ---: | ---: | ---: | --- |",
    ]
    for row in rows:
        lines.append(
            f"| `{row['scenarioID']}` | {row['baselineMs']:.3f} | {row['candidateMs']:.3f} "
            f"| {row['deltaPct']:+.2f}% | {row['baselineSpreadPct']:.1f}% | {row['candidateSpreadPct']:.1f}% "
            f"| {'yes' if row['regressedBeyondNoise'] else 'no'} |"
        )
    args.markdown.parent.mkdir(parents=True, exist_ok=True)
    args.markdown.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(json.dumps(result, indent=2))
    return 0 if gate_passed else 1


if __name__ == "__main__":
    raise SystemExit(main())
