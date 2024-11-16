package main

import (
	"testing"
)

func BenchmarkIPAndPort(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ip := uint32(0)
		port := uint16(25565)

		// A limit of how many iterations it should do
		j := 1000000
		for {
			if j == 0 {
				break
			}
			j--

			address := parseAddress(ip, port)
			if address == "" {
				b.Fail()
			}

			ip++
		}
	}
}
