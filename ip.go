package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Parse an IP string into its segments.
func parseIP(ip string, segA *uint8, segB *uint8, segC *uint8, segD *uint8) error {
	octets := strings.Split(ip, ".")
	if len(octets) != 4 {
		return fmt.Errorf("invalid IP address '%s'", ip)
	}

	newSegA, err := strconv.ParseUint(octets[0], 10, 8)
	if err != nil {
		return fmt.Errorf("invalid IP address '%s', %s", ip, err)
	}

	newSegB, err := strconv.ParseUint(octets[1], 10, 8)
	if err != nil {
		return fmt.Errorf("invalid IP address '%s', %s", ip, err)
	}

	newSegC, err := strconv.ParseUint(octets[2], 10, 8)
	if err != nil {
		return fmt.Errorf("invalid IP address '%s', %s", ip, err)
	}

	newSegD, err := strconv.ParseUint(octets[3], 10, 8)
	if err != nil {
		return fmt.Errorf("invalid IP address '%s', %s", ip, err)
	}

	*segA = uint8(newSegA)
	*segB = uint8(newSegB)
	*segC = uint8(newSegC)
	*segD = uint8(newSegD)

	return nil
}

// Check if the IP is in any reserved range, skips to the next available range if it is.
func isReserved(segA *uint8, segB *uint8, segC *uint8) bool {
	// 10.x.x.x
	// 127.x.x.x
	if *segA == 10 || *segA == 127 {
		*segA++
		return true
	}

	// 169.254.x.x
	if *segA == 169 && *segB == 254 {
		*segB = 255
		return true
	}

	// 172.(>= 16 && <= 31).x.x
	if *segA == 172 && *segB >= 16 && *segB <= 31 {
		*segB = 32
		return true
	}

	if *segA == 192 {
		if *segB == 0 {
			// 192.0.0.x
			// 192.0.2.x
			if *segC == 0 || *segC == 2 {
				*segC++
				return true
			}

			return false
		}

		// 192.88.99.0
		if *segB == 88 && *segC == 99 {
			*segC = 100
			return true
		}

		// 192.168.x.x
		if *segB == 168 {
			*segB = 169
			return true
		}

		return false
	}

	// 198.51.100.x
	if *segA == 198 && *segB == 51 && *segC == 100 {
		*segC = 101
		return true
	}

	// 203.0.113.x
	if *segA == 203 && *segB == 0 && *segC == 113 {
		*segC = 114
		return true
	}

	return false
}
