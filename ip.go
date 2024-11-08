package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Converts an IP and port to a string
func parseAddress(ip uint32, port uint16) string {
	var address [21]byte
	i := 0

	// Helper function to write a 3-digit segment into the buffer
	appendSegment := func(segment byte) {
		if segment >= 100 {
			address[i] = '0' + segment/100
			i++
			segment %= 100
		}

		if segment >= 10 {
			address[i] = '0' + segment/10
			i++
			segment %= 10
		}

		address[i] = '0' + segment
		i++
	}

	appendSegment(byte(ip >> 24))
	address[i] = '.'
	i++

	appendSegment(byte(ip >> 16))
	address[i] = '.'
	i++

	appendSegment(byte(ip >> 8))
	address[i] = '.'
	i++

	appendSegment(byte(ip))

	address[i] = ':'
	i++

	start := i
	if port >= 10000 {
		address[i] = '0' + byte(port/10000)
		i++
		port %= 10000
	}

	if port >= 1000 || i > start {
		address[i] = '0' + byte(port/1000)
		i++
		port %= 1000
	}

	if port >= 100 || i > start {
		address[i] = '0' + byte(port/100)
		i++
		port %= 100
	}

	if port >= 10 || i > start {
		address[i] = '0' + byte(port/10)
		i++
		port %= 10
	}

	address[i] = '0' + byte(port)
	i++

	return string(address[:i])
}

// Converts an IP (x.x.x.x) string to an uint32
func parseIP(ip string) (uint32, error) {
	if ip == "" {
		return 1 << 24, nil
	}

	octets := strings.Split(ip, ".")
	if len(octets) != 4 {
		return 0, fmt.Errorf("invalid IP address '%s'", ip)
	}

	segA, err := strconv.Atoi(octets[0])
	if err != nil || segA < 0 || segA > 255 {
		return 0, fmt.Errorf("invalid segment '%s' in IP '%s'", octets[0], ip)
	}

	segB, err := strconv.Atoi(octets[1])
	if err != nil || segB < 0 || segB > 255 {
		return 0, fmt.Errorf("invalid segment '%s' in IP '%s'", octets[1], ip)
	}

	segC, err := strconv.Atoi(octets[2])
	if err != nil || segC < 0 || segC > 255 {
		return 0, fmt.Errorf("invalid segment '%s' in IP '%s'", octets[2], ip)
	}

	segD, err := strconv.Atoi(octets[3])
	if err != nil || segD < 0 || segD > 255 {
		return 0, fmt.Errorf("invalid segment '%s' in IP '%s'", octets[3], ip)
	}

	parsed := (uint32(segA) << 24) | (uint32(segB) << 16) | (uint32(segC) << 8) | uint32(segD)
	return parsed, nil
}

// Check if the IP is in any reserved range, skips to the next available range if it is.
func isReserved(ip *uint32) bool {
	segA := (*ip >> 24) & 0xFF
	segB := (*ip >> 16) & 0xFF
	segC := (*ip >> 8) & 0xFF

	// 10.x.x.x
	// 127.x.x.x
	if segA == 10 || segA == 127 {
		*ip += 1 << 24
		return true
	}

	// 169.254.x.x
	if segA == 169 && segB == 254 {
		*ip += 1 << 16
		return true
	}

	// 172.(>= 16 && <= 31).x.x
	if segA == 172 && segB >= 16 && segB <= 31 {
		*ip += (32 - segB) << 16 // Move B segment to 32
		return true
	}

	if segA == 192 {
		if segB == 0 {
			// 192.0.0.x
			// 192.0.2.x
			if segC == 0 || segC == 2 {
				*ip += 1 << 8
				return true
			}
			return false
		}

		// 192.88.99.0
		if segB == 88 && segC == 99 {
			*ip += 1 << 8 // Move C segment to 100
			return true
		}

		// 192.168.x.x
		if segB == 168 {
			*ip += 1 << 16
			return true
		}

		return false
	}

	// 198.51.100.x
	if segA == 198 && segB == 51 && segC == 100 {
		*ip += 1 << 8 // Move C segment to 101
		return true
	}

	// 203.0.113.x
	if segA == 203 && segB == 0 && segC == 113 {
		*ip += 1 << 8 // Move C segment to 114
		return true
	}

	return false
}
