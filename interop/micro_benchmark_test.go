package interop

import "testing"

func BenchmarkClientCanExchangeMicro(parseB *testing.B) {
	parseLocal := ClientCapabilities{
		ProtocolVersion: "1.2",
		Encodings:       []string{"json", "binary"},
		Topics:          []string{"orders", "clients", "presence"},
		MaxJSONBytes:    1 << 20,
		MaxBinaryBytes:  1 << 20,
	}
	parsePeer := ClientCapabilities{
		ProtocolVersion: "1.2",
		Encodings:       []string{"json"},
		Topics:          []string{"orders", "clients"},
		MaxJSONBytes:    1 << 18,
		MaxBinaryBytes:  1 << 18,
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = ClientCanExchange(parseLocal, parsePeer, "orders", ClientPayloadJSON)
	}
}
