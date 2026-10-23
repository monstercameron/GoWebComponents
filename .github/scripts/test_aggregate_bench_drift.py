#!/usr/bin/env python3
"""Focused CLI tests for ABBA benchmark report aggregation."""

import copy
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name("aggregate-bench-drift.py")


def buildReport(parseSamples, parseRunOrder=None):
    """buildReport returns a minimal suite report containing one benchmark."""
    parseMedian = sum(parseSamples) / len(parseSamples)
    return {
        "schemaVersion": "gwc.bench.suite/v1",
        "suite": "test",
        "generatedAt": "2026-10-23T00:00:00Z",
        "machine": {"os": "linux", "arch": "amd64", "cpuModel": "synthetic"},
        "run": {"count": len(parseSamples), "warmup": 1, "seed": 7, "order": parseRunOrder or ["./a"], "parallel": 1},
        "lanes": [{
            "lane": "native",
            "layer": "native",
            "status": "ok",
            "entries": [{
                "id": "native/./a/BenchmarkX/sub",
                "unit": "ns/op",
                "samples": parseSamples,
                "median": parseMedian,
                "mad": 0,
                "n": len(parseSamples),
                "bytesPerOp": 10,
                "allocsPerOp": 2,
            }],
        }],
        "targets": [{"lane": "native", "package": "./a", "status": "ok"}],
        "size": [],
        "refs": {},
        "health": {},
    }


class AggregateBenchDriftTest(unittest.TestCase):
    """Verify the aggregator combines samples and rejects unsafe input."""

    def runAggregator(self, parseFirst, parseSecond):
        """runAggregator writes input reports to a temporary directory and invokes the CLI."""
        parseTemp = tempfile.TemporaryDirectory()
        self.addCleanup(parseTemp.cleanup)
        parseRoot = Path(parseTemp.name)
        parsePaths = [parseRoot / "first.json", parseRoot / "second.json"]
        for parsePath, parseReport in zip(parsePaths, (parseFirst, parseSecond)):
            parsePath.write_text(json.dumps(parseReport), encoding="utf-8")
        parseOutput = parseRoot / "merged.json"
        parseResult = subprocess.run(
            [sys.executable, str(SCRIPT), "--out", str(parseOutput), "--suite", "test-abba", *(str(path) for path in parsePaths)],
            text=True,
            capture_output=True,
            check=False,
        )
        return parseResult, parseOutput

    def testAggregatesRawSamplesAndRunMetadata(self):
        """testAggregatesRawSamplesAndRunMetadata recomputes robust summaries and metrics."""
        parseFirst = buildReport([100.0, 102.0], ["./a"])
        parseSecond = buildReport([300.0, 104.0], ["./a"])
        parseSecond["lanes"][0]["entries"][0]["bytesPerOp"] = 30
        parseSecond["lanes"][0]["entries"][0]["allocsPerOp"] = 4
        parseResult, parseOutput = self.runAggregator(parseFirst, parseSecond)
        self.assertEqual(parseResult.returncode, 0, parseResult.stderr)
        parseMerged = json.loads(parseOutput.read_text(encoding="utf-8"))
        parseEntry = parseMerged["lanes"][0]["entries"][0]
        self.assertEqual(parseEntry["samples"], [100.0, 102.0, 300.0, 104.0])
        self.assertEqual((parseEntry["median"], parseEntry["mad"], parseEntry["n"]), (103.0, 2.0, 4))
        self.assertEqual((parseEntry["bytesPerOp"], parseEntry["allocsPerOp"]), (20.0, 3.0))
        self.assertEqual(parseMerged["run"]["order"], ["./a", "./a"])
        self.assertEqual((parseMerged["run"]["count"], parseMerged["run"]["warmup"]), (4, 2))

    def testRejectsDifferentBenchmarkCoverage(self):
        """testRejectsDifferentBenchmarkCoverage rejects an entry missing from one report."""
        parseFirst = buildReport([100.0, 102.0])
        parseSecond = copy.deepcopy(parseFirst)
        parseSecond["lanes"][0]["entries"][0]["id"] = "native/./a/BenchmarkY/sub"
        parseResult, _ = self.runAggregator(parseFirst, parseSecond)
        self.assertNotEqual(parseResult.returncode, 0)
        self.assertIn("coverage differs", parseResult.stderr)

    def testRejectsIncompleteSamples(self):
        """testRejectsIncompleteSamples rejects a report with fewer samples than run.count."""
        parseFirst = buildReport([100.0, 102.0])
        parseSecond = buildReport([103.0, 104.0])
        parseSecond["lanes"][0]["entries"][0]["samples"] = [103.0]
        parseResult, _ = self.runAggregator(parseFirst, parseSecond)
        self.assertNotEqual(parseResult.returncode, 0)
        self.assertIn("incomplete samples", parseResult.stderr)


if __name__ == "__main__":
    unittest.main()
