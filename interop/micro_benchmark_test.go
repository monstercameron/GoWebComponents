package interop

import "testing"

func BenchmarkClientCanExchangeMicro(b *testing.B) {
	local := ClientCapabilities{
		ProtocolVersion: "1.2",
		Encodings:       []string{"json", "binary"},
		Topics:          []string{"orders", "clients", "presence"},
		MaxJSONBytes:    1 << 20,
		MaxBinaryBytes:  1 << 20,
	}
	peer := ClientCapabilities{
		ProtocolVersion: "1.2",
		Encodings:       []string{"json"},
		Topics:          []string{"orders", "clients"},
		MaxJSONBytes:    1 << 18,
		MaxBinaryBytes:  1 << 18,
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = ClientCanExchange(local, peer, "orders", ClientPayloadJSON)
	}
}
