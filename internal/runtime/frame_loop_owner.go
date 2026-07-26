package runtime

import goruntime "runtime"

// Frame-loop ownership (v5 P2.1).
//
// insideFrameLoop originally asked "is a frame-loop region on the stack", using
// per-runtime depth counters. That question has a wrong answer for the case the
// inbox exists to serve.
//
// A handler that spawns a goroutine — the documented way to call a worker
// command without deadlocking the event loop — leaves that goroutine running
// while the handler is still on the stack and the depth is still non-zero. The
// setter it calls therefore read "inside the frame loop" and applied directly,
// at an arbitrary moment relative to the in-flight tree. The mechanism missed
// its own motivating case, and missed it silently: an app that enabled
// AsyncIngress and followed the documented pattern got no isolation and no
// indication it had none.
//
// The question has to be "am I on the goroutine that entered the frame loop",
// which needs an identity Go does not expose. Parsing it out of the stack header
// is the standard workaround and the one the hook threading guard already uses.
//
// It is NOT that function. computeHookGoroutineID is compiled to a constant 0 in
// production builds, because the hook guard runs per hook call — thousands per
// frame — and could not carry the cost. This runs per state write, which is
// orders of magnitude rarer, and a guarantee that evaporates in production
// builds is not a guarantee.
//
// Cost, measured: 1.15 µs per call. Paid only when AsyncIngress is enabled, and
// only on writes that would otherwise be applied directly, so a default build
// pays nothing. Against a 16 ms frame, twenty writes cost 0.14%.

// frameLoopGoroutineID returns an identifier for the calling goroutine.
//
// The 64-byte buffer is deliberate: runtime.Stack fills what it is given and the
// id sits in the first line, so a small buffer avoids formatting frames that
// would be discarded. Returns 0 if the header cannot be read, and callers treat
// 0 as "unknown", which fails safe — an unknown identity is never taken as
// matching the owner.
func frameLoopGoroutineID() uint64 {
	var parseBuffer [64]byte
	parseCount := goruntime.Stack(parseBuffer[:], false)
	const parsePrefix = "goroutine "
	if parseCount <= len(parsePrefix) {
		return 0
	}
	parseID := uint64(0)
	parseDigits := 0
	for _, parseByte := range parseBuffer[len(parsePrefix):parseCount] {
		if parseByte < '0' || parseByte > '9' {
			break
		}
		parseID = parseID*10 + uint64(parseByte-'0')
		parseDigits++
	}
	if parseDigits == 0 {
		return 0
	}
	return parseID
}
