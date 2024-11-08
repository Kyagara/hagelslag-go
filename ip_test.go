package main

import (
	"testing"
)

func BenchmarkIPToString(b *testing.B) {
	b.ReportAllocs()

	for s := 0; s < b.N; s++ {
		i := uint32(0)
		g := 10000000

		for {
			if g == 0 {
				break
			}
			g--

			ip := ipFromUint32(i)
			if ip == "" {
				b.Fail()
			}

			i++
		}
	}
}
