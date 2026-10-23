#!/usr/bin/env python3
"""Combine matching native benchmark samples from one ABBA pair of reports."""

import argparse
import json
import math
import statistics
from pathlib import Path


def runMain():
    """runMain combines two same-side reports into a raw-sample aggregate."""
    parseParser = argparse.ArgumentParser(description=__doc__)
    parseParser.add_argument("--out", required=True, type=Path)
    parseParser.add_argument("--suite", required=True)
    parseParser.add_argument("reports", nargs=2, type=Path)
    parseArgs = parseParser.parse_args()

    loadReports = [json.loads(path.read_text(encoding="utf-8")) for path in parseArgs.reports]
    parseFirst, parseSecond = loadReports
    if parseFirst.get("schemaVersion") != "gwc.bench.suite/v1" or parseSecond.get("schemaVersion") != parseFirst.get("schemaVersion"):
        raise SystemExit("reports must use the same supported bench suite schema")
    if parseFirst.get("machine") != parseSecond.get("machine"):
        raise SystemExit("reports use different benchmark machines")
    if parseFirst.get("run", {}).get("count") != parseSecond.get("run", {}).get("count"):
        raise SystemExit("reports use different per-run sample counts")
    for parseKey in ("seed", "benchtime", "parallel"):
        if parseFirst.get("run", {}).get(parseKey) != parseSecond.get("run", {}).get(parseKey):
            raise SystemExit(f"reports use different run {parseKey}")

    def getNativeEntries(parseReport):
        """getNativeEntries validates and indexes one report's native data."""
        parseLanes = [parseLane for parseLane in parseReport.get("lanes", []) if parseLane.get("lane") == "native"]
        if len(parseLanes) != 1 or parseLanes[0].get("status") != "ok":
            raise SystemExit("each report must contain one successful native lane")
        parseTargets = parseReport.get("targets", [])
        if not parseTargets or any(parseTarget.get("lane") != "native" or parseTarget.get("status") != "ok" for parseTarget in parseTargets):
            raise SystemExit("each report must contain only successful native targets")
        parseTargetKeys = [(parseTarget.get("lane"), parseTarget.get("package")) for parseTarget in parseTargets]
        if len(set(parseTargetKeys)) != len(parseTargetKeys):
            raise SystemExit("native lane contains duplicate target IDs")
        parseEntries = parseLanes[0].get("entries", [])
        parseIndexed = {parseEntry.get("id"): parseEntry for parseEntry in parseEntries}
        if len(parseIndexed) != len(parseEntries) or not parseIndexed:
            raise SystemExit("native lane has no entries or contains duplicate benchmark IDs")
        return parseLanes[0], set(parseTargetKeys), parseIndexed

    parseFirstLane, parseFirstTargets, parseFirstEntries = getNativeEntries(parseFirst)
    _, parseSecondTargets, parseSecondEntries = getNativeEntries(parseSecond)
    if parseFirstTargets != parseSecondTargets:
        raise SystemExit("native target coverage differs between reports")
    if parseFirstEntries.keys() != parseSecondEntries.keys():
        parseMissing = sorted(parseFirstEntries.keys() ^ parseSecondEntries.keys())
        raise SystemExit("native benchmark coverage differs: " + ", ".join(parseMissing[:10]))

    for parseBenchmarkID, parseEntry in parseFirstEntries.items():
        parseOther = parseSecondEntries[parseBenchmarkID]
        if parseEntry.get("unit") != parseOther.get("unit"):
            raise SystemExit(f"unit differs for {parseBenchmarkID}")
        parseLeftSamples = parseEntry.get("samples", [])
        parseRightSamples = parseOther.get("samples", [])
        parseSamples = parseLeftSamples + parseRightSamples
        if len(parseLeftSamples) != parseFirst["run"]["count"] or len(parseRightSamples) != parseSecond["run"]["count"]:
            raise SystemExit(f"incomplete samples for {parseBenchmarkID}")
        if not parseSamples or any(not math.isfinite(value) for value in parseSamples):
            raise SystemExit(f"missing or invalid raw samples for {parseBenchmarkID}")
        parseMedian = statistics.median(parseSamples)
        parseEntry["samples"] = parseSamples
        parseEntry["median"] = parseMedian
        parseEntry["mad"] = statistics.median(abs(value - parseMedian) for value in parseSamples)
        parseEntry["n"] = len(parseSamples)
        for parseMetric in ("bytesPerOp", "allocsPerOp"):
            parseLeftMetric = parseEntry.get(parseMetric)
            parseRightMetric = parseOther.get(parseMetric)
            if (parseLeftMetric is None) != (parseRightMetric is None):
                raise SystemExit(f"incomplete {parseMetric} for {parseBenchmarkID}")
            if parseLeftMetric is None:
                parseEntry.pop(parseMetric, None)
            elif not math.isfinite(parseLeftMetric) or not math.isfinite(parseRightMetric):
                raise SystemExit(f"invalid {parseMetric} for {parseBenchmarkID}")
            else:
                parseEntry[parseMetric] = (
                    parseLeftMetric * len(parseLeftSamples)
                    + parseRightMetric * len(parseRightSamples)
                ) / len(parseSamples)
        parseEntry.pop("sourceReport", None)

    parseFirst["suite"] = parseArgs.suite
    parseFirst["run"]["count"] += parseSecond["run"]["count"]
    parseFirst["run"]["warmup"] += parseSecond["run"].get("warmup", 0)
    parseFirst["run"]["order"] = parseFirst["run"].get("order", []) + parseSecond["run"].get("order", [])
    parseFirst["lanes"] = [parseFirstLane]
    parseFirst.setdefault("health", {})["aggregation"] = {
        "sources": [parsePath.name for parsePath in parseArgs.reports],
        "method": "raw-sample-concatenation",
        "scalarMetrics": "sample-weighted-mean",
    }
    parseArgs.out.write_text(json.dumps(parseFirst, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    runMain()
