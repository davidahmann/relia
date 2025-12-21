package crypto

import "testing"

func BenchmarkCanonicalize(b *testing.B) {
	input := map[string]any{
		"schema": "bench",
		"nested": map[string]any{
			"b": "two",
			"a": "one",
			"n": 123,
		},
		"list": []any{3, 2, 1, "x"},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Canonicalize(input); err != nil {
			b.Fatalf("canonicalize: %v", err)
		}
	}
}
