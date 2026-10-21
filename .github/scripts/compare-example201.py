#!/usr/bin/env python3
"""Compare two Example 201 browser benchmark reports.

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
        value = item.get("getDomReadyRepresentativeMs")
        if not isinstance(value, (int, float)) or not math.isfinite(value) or value <= 0:
            raise ValueError(f"{path}: {scenario_id} has invalid representative timing {value!r}")
        result[scenario_id] = item
    return payload, report, result


def geometric_mean(values: list[float]) -> float:
    return math.exp(sum(math.log(value) for value in values) / len(values))


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("baseline", type=Path)
    parser.add_argument("candidate", type=Path)
    parser.add_argument("--json", type=Path, required=True)
    parser.add_argument("--markdown", type=Path, required=True)
    parser.add_argument("--max-regress-pct", type=float, default=30.0)
    args = parser.parse_args()

    try:
        baseline_payload, baseline_report, baseline = load_runtime_report(args.baseline)
        candidate_payload, candidate_report, candidate = load_runtime_report(args.candidate)
        if baseline_payload.get("getBrowser") != candidate_payload.get("getBrowser"):
            raise ValueError("baseline and candidate use different browsers")
        if baseline_report.get("getScenarioOrder") != candidate_report.get("getScenarioOrder"):
            raise ValueError("baseline and candidate scenario orders differ")
        if baseline_report.get("getSeed") != candidate_report.get("getSeed"):
            raise ValueError("baseline and candidate seeds differ")
        if set(baseline) != set(candidate):
            raise ValueError("baseline and candidate scenario sets differ")
        if not math.isfinite(args.max_regress_pct) or args.max_regress_pct < 0:
            raise ValueError("--max-regress-pct must be finite and non-negative")
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"Example 201 comparison error: {exc}", file=sys.stderr)
        return 2

    rows = []
    for scenario_id in baseline_report["getScenarioOrder"]:
        if scenario_id not in baseline:
            continue
        baseline_ms = float(baseline[scenario_id]["getDomReadyRepresentativeMs"])
        candidate_ms = float(candidate[scenario_id]["getDomReadyRepresentativeMs"])
        delta_pct = (candidate_ms / baseline_ms - 1.0) * 100.0
        rows.append({
            "scenarioID": scenario_id,
            "baselineMs": baseline_ms,
            "candidateMs": candidate_ms,
            "deltaPct": delta_pct,
            "regressedBeyondNoise": delta_pct > args.max_regress_pct,
        })
    baseline_mean = geometric_mean([row["baselineMs"] for row in rows])
    candidate_mean = geometric_mean([row["candidateMs"] for row in rows])
    geomean_delta_pct = (candidate_mean / baseline_mean - 1.0) * 100.0
    failures = [row["scenarioID"] for row in rows if row["regressedBeyondNoise"]]
    gate_passed = not failures and geomean_delta_pct <= 0.0
    result = {
        "schema": "example201-v6-v7/v1",
        "baseline": {"path": str(args.baseline), "generatedAt": baseline_payload.get("getGeneratedAt")},
        "candidate": {"path": str(args.candidate), "generatedAt": candidate_payload.get("getGeneratedAt")},
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
        f"- Scenarios: `{result['scenarioCount']}`",
        f"- Noise floor: `+{args.max_regress_pct:.1f}%` per scenario",
        f"- Baseline geomean: `{baseline_mean:.3f} ms`",
        f"- Candidate geomean: `{candidate_mean:.3f} ms`",
        f"- Geomean delta: `{geomean_delta_pct:+.2f}%`",
        f"- Gate: **{'PASS' if gate_passed else 'FAIL'}**",
        "",
        "| Scenario | v6 DOM Ready Rep. (ms) | v7 DOM Ready Rep. (ms) | Delta | Beyond noise |",
        "| --- | ---: | ---: | ---: | --- |",
    ]
    for row in rows:
        lines.append(
            f"| `{row['scenarioID']}` | {row['baselineMs']:.3f} | {row['candidateMs']:.3f} "
            f"| {row['deltaPct']:+.2f}% | {'yes' if row['regressedBeyondNoise'] else 'no'} |"
        )
    args.markdown.parent.mkdir(parents=True, exist_ok=True)
    args.markdown.write_text("\n".join(lines) + "\n", encoding="utf-8")
    print(json.dumps(result, indent=2))
    return 0 if gate_passed else 1


if __name__ == "__main__":
    raise SystemExit(main())
