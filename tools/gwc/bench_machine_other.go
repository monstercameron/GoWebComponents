//go:build !windows

package main

import (
	"os"
	"strconv"
	"strings"
)

// readBenchmarkCPUModel reads the CPU model from /proc/cpuinfo or sysctl, best effort.
func readBenchmarkCPUModel() string {
	if parseData, parseErr := os.ReadFile("/proc/cpuinfo"); parseErr == nil {
		for _, parseLine := range strings.Split(string(parseData), "\n") {
			for _, parseKey := range []string{"model name", "Model", "Hardware"} {
				if strings.HasPrefix(parseLine, parseKey) {
					if _, parseValue, isFound := strings.Cut(parseLine, ":"); isFound {
						return strings.TrimSpace(parseValue)
					}
				}
			}
		}
	}
	return benchmarkQueryCommand("", "sysctl", "-n", "machdep.cpu.brand_string")
}

// readBenchmarkRAMBytes reads total memory from /proc/meminfo or sysctl, best effort.
func readBenchmarkRAMBytes() uint64 {
	if parseData, parseErr := os.ReadFile("/proc/meminfo"); parseErr == nil {
		for _, parseLine := range strings.Split(string(parseData), "\n") {
			if parseRest, isFound := strings.CutPrefix(parseLine, "MemTotal:"); isFound {
				parseFields := strings.Fields(parseRest)
				if len(parseFields) > 0 {
					if parseKb, parseErr2 := strconv.ParseUint(parseFields[0], 10, 64); parseErr2 == nil {
						return parseKb * 1024
					}
				}
			}
		}
	}
	if parseValue, parseErr := strconv.ParseUint(benchmarkQueryCommand("", "sysctl", "-n", "hw.memsize"), 10, 64); parseErr == nil {
		return parseValue
	}
	return 0
}
