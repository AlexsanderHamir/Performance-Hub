package parser

import (
	"crypto/sha256"
	"testing"
)

// work does CPU-bound work so the benchmark produces a useful CPU profile.
func work(n int) {
	h := sha256.New()
	b := make([]byte, 1024)
	for i := 0; i < n; i++ {
		b[0] = byte(i)
		h.Write(b)
		h.Sum(b[:0])
	}
}

func BenchmarkWork(b *testing.B) {
	for i := 0; i < b.N; i++ {
		work(1000)
	}
}

func BenchmarkWorkHeavy(b *testing.B) {
	for i := 0; i < b.N; i++ {
		work(5000)
	}
}
