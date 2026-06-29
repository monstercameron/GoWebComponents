package pluginruntime

// BackendID identifies one implementation backend normalized by the kernel.
type BackendID string

const (
	BackendIDRuntime1 BackendID = "runtime1"
	BackendIDRuntime2 BackendID = "runtime2"
	BackendIDNative   BackendID = "native"
)

// BuildSnapshotMeta returns one normalized snapshot metadata value.
func BuildSnapshotMeta(parseBackend BackendID, isParseTruncated bool) SnapshotMeta {
	return SnapshotMeta{
		BackendID: string(parseBackend),
		Truncated: isParseTruncated,
	}
}
