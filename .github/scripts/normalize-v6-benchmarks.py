#!/usr/bin/env python3
"""Convert the Go 1.23-era b.N benchmark loops in the v6 source tree.

The v7 benchmark suite uses testing.B.Loop.  Applying the same loop driver to
the historical source keeps the v6/v7 comparison focused on the implementation.
The old loop index is retained because some benchmarks deliberately vary input
with the iteration number.  It remains in the loop's original scope and keeps
its original zero-based values by using the post statement.
"""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path


LOOP = re.compile(
    r"(?P<indent>^[ \t]*)for[ \t]+(?P<index>[A-Za-z_]\w*)[ \t]*:=[ \t]*0;"
    r"[ \t]*(?P=index)[ \t]*<[ \t]*(?P<bench>[A-Za-z_]\w*)\.N;"
    r"[ \t]*(?P=index)\+\+[ \t]*\{",
    re.MULTILINE,
)


def normalize_props_storage_source(path: Path, source: str) -> tuple[str, int]:
    """Make the prototype's map-to-slice benchmark deterministic in v6 too."""
    if not is_runtime_benchmark(path, "props_storage_prototype_test.go") or "func clonePropsToSlice" not in source:
        return source, 0
    parse_insertions = 0
    if '"sort"' not in source:
        source = source.replace('import "testing"', 'import (\n\t"sort"\n\t"testing"\n)', 1)
    marker = "\tfor parseKey, parseValue := range parseProps {\n\t\tgetPairs = append(getPairs, propPair{Key: parseKey, Value: parseValue})\n\t}\n"
    replacement = marker + "\tsort.Slice(getPairs, func(parseI, parseJ int) bool { return getPairs[parseI].Key < getPairs[parseJ].Key })\n"
    if marker in source and "sort.Slice(getPairs" not in source:
        source = source.replace(marker, replacement, 1)
        parse_insertions = 1
    return source, parse_insertions


def normalize_hydration_scheduler_source(path: Path, source: str) -> tuple[str, int]:
    """Run the same complete scheduler drain on the historical hydration benchmarks."""
    if not is_runtime_benchmark(path, "hydration_benchmark_test.go"):
        return source, 0
    parse_replacements = source.count("parseScheduler.timeouts[0]()")
    return source.replace("parseScheduler.timeouts[0]()", "drainBenchmarkScheduler(parseScheduler)"), parse_replacements


def is_runtime_benchmark(path: Path, name: str) -> bool:
    return path.name == name and path.parent.name == "runtime" and path.parent.parent.name == "internal"


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("root", type=Path)
    parser.add_argument("--report", type=Path, required=True)
    args = parser.parse_args()

    files: list[dict[str, object]] = []
    transformations: list[dict[str, object]] = []
    old_loops = 0
    replacements = 0
    for path in sorted(args.root.rglob("*_test.go")):
        before = path.read_text(encoding="utf-8")
        matches = list(LOOP.finditer(before))
        old_loops += len(matches)

        def replace(match: re.Match[str]) -> str:
            indent = match.group("indent")
            index = match.group("index")
            bench = match.group("bench")
            return f"{indent}for {index} := 0; {bench}.Loop(); {index}++ {{"

        after = LOOP.sub(replace, before)
        after, parse_props_insertions = normalize_props_storage_source(path, after)
        after, parse_hydration_replacements = normalize_hydration_scheduler_source(path, after)
        if is_runtime_benchmark(path, "props_storage_prototype_test.go"):
            transformations.append({"path": path.relative_to(args.root).as_posix(), "props_sort_insertions": parse_props_insertions})
            if "sort.Slice(getPairs" not in after:
                raise SystemExit(f"props storage benchmark parity marker missing: {path}")
        if is_runtime_benchmark(path, "hydration_benchmark_test.go"):
            transformations.append({"path": path.relative_to(args.root).as_posix(), "hydration_drain_replacements": parse_hydration_replacements})
            if "parseScheduler.timeouts[0]()" in after or "drainBenchmarkScheduler(parseScheduler)" not in after:
                raise SystemExit(f"hydration scheduler parity marker missing: {path}")
        if not matches and after == before:
            continue
        path.write_text(after, encoding="utf-8")
        replacements += len(matches)
        files.append({"path": str(path.relative_to(args.root)), "count": len(matches)})

    remaining = sum(len(LOOP.findall(path.read_text(encoding="utf-8"))) for path in args.root.rglob("*_test.go"))
    transformed_paths = {str(parseEntry["path"]) for parseEntry in transformations}
    expected_paths = {"internal/runtime/props_storage_prototype_test.go", "internal/runtime/hydration_benchmark_test.go"}
    missing_paths = sorted(expected_paths - transformed_paths)
    if missing_paths:
        raise SystemExit("expected benchmark parity sources missing: " + ", ".join(missing_paths))
    report = {
        "root": str(args.root),
        "old_loop_count": old_loops,
        "replacement_count": replacements,
        "remaining_old_loop_count": remaining,
        "files": files,
        "transformations": transformations,
    }
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.report.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, indent=2))
    if replacements != old_loops or remaining:
        raise SystemExit("v6 benchmark loop normalization was incomplete")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
