package uri

import "testing"

// BenchmarkParseBatch10k reports ns/op for one 10k-URI batch; configs/sec
// = 10000 / ns_per_op * 1e9.
func BenchmarkParseBatch10k(b *testing.B) {
	uris := BenchURIs(10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, u := range uris {
			if _, err := Parse(u); err != nil {
				b.Fatal(err)
			}
		}
	}
}
