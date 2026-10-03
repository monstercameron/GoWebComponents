//go:build windows

package main

import (
	"strings"
	"syscall"
	"unsafe"
)

// readBenchmarkCPUModel reads the processor name from the registry via reg.exe.
func readBenchmarkCPUModel() string {
	parseOut := benchmarkQueryCommand("", "reg", "query", `HKLM\HARDWARE\DESCRIPTION\System\CentralProcessor\0`, "/v", "ProcessorNameString")
	for _, parseLine := range strings.Split(parseOut, "\n") {
		parseFields := strings.SplitN(strings.TrimSpace(parseLine), "REG_SZ", 2)
		if len(parseFields) == 2 && strings.HasPrefix(parseFields[0], "ProcessorNameString") {
			return strings.TrimSpace(parseFields[1])
		}
	}
	return ""
}

type benchmarkMemoryStatus struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

// readBenchmarkRAMBytes returns total physical memory via kernel32 GlobalMemoryStatusEx.
func readBenchmarkRAMBytes() uint64 {
	parseProc := syscall.NewLazyDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")
	parseStatus := benchmarkMemoryStatus{}
	parseStatus.Length = uint32(unsafe.Sizeof(parseStatus))
	parseRet, _, _ := parseProc.Call(uintptr(unsafe.Pointer(&parseStatus)))
	if parseRet == 0 {
		return 0
	}
	return parseStatus.TotalPhys
}
