package main

import (
	"testing"
)

func BenchmarkIPToString(b *testing.B) {
	b.ReportAllocs()

	for s := 0; s < b.N; s++ {
		i := uint32(0)
		port := uint16(25565)
		g := 1000000

		for {
			if g == 0 {
				break
			}
			g--

			ip := parseAddress(i, port)
			if ip == "" {
				b.Fail()
			}

			i++
		}
	}
}
