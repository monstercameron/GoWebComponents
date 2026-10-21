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


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("root", type=Path)
    parser.add_argument("--report", type=Path, required=True)
    args = parser.parse_args()

    files: list[dict[str, object]] = []
    old_loops = 0
    replacements = 0
    for path in sorted(args.root.rglob("*_test.go")):
        before = path.read_text(encoding="utf-8")
        matches = list(LOOP.finditer(before))
        if not matches:
            continue
        old_loops += len(matches)

        def replace(match: re.Match[str]) -> str:
            indent = match.group("indent")
            index = match.group("index")
            bench = match.group("bench")
            return f"{indent}for {index} := 0; {bench}.Loop(); {index}++ {{"

        after = LOOP.sub(replace, before)
        path.write_text(after, encoding="utf-8")
        replacements += len(matches)
        files.append({"path": str(path.relative_to(args.root)), "count": len(matches)})

    remaining = sum(len(LOOP.findall(path.read_text(encoding="utf-8"))) for path in args.root.rglob("*_test.go"))
    report = {
        "root": str(args.root),
        "old_loop_count": old_loops,
        "replacement_count": replacements,
        "remaining_old_loop_count": remaining,
        "files": files,
    }
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.report.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, indent=2))
    if replacements != old_loops or remaining:
        raise SystemExit("v6 benchmark loop normalization was incomplete")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
