package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Parse an IP string into its segments.
func parseIP(ip string, seg_a *uint8, seg_b *uint8, seg_c *uint8, seg_d *uint8) error {
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

	*seg_a = uint8(newSegA)
	*seg_b = uint8(newSegB)
	*seg_c = uint8(newSegC)
	*seg_d = uint8(newSegD)

	return nil
}

// Check if the IP is in any reserved range, skips to the next available range if it is.
func isReserved(seg_a *uint8, seg_b *uint8, seg_c *uint8) bool {
	// 10.x.x.x
	// 127.x.x.x
	if *seg_a == 10 || *seg_a == 127 {
		*seg_a = *seg_a + 1
		return true
	}

	// 169.254.x.x
	if *seg_a == 169 && *seg_b == 254 {
		*seg_b = 255
		return true
	}

	// 172.(>= 16 && <= 31).x.x
	if *seg_a == 172 && *seg_b >= 16 && *seg_b <= 31 {
		*seg_b = 32
		return true
	}

	if *seg_a == 192 {
		if *seg_b == 0 {
			// 192.0.0.x
			// 192.0.2.x
			if *seg_c == 0 || *seg_c == 2 {
				*seg_c = *seg_c + 1
				return true
			}

			return false
		}

		// 192.88.99.0
		if *seg_b == 88 && *seg_c == 99 {
			*seg_c = 100
			return true
		}

		// 192.168.x.x
		if *seg_b == 168 {
			*seg_b = 169
			return true
		}

		return false
	}

	// 198.51.100.x
	if *seg_a == 198 && *seg_b == 51 && *seg_c == 100 {
		*seg_c = 101
		return true
	}

	// 203.0.113.x
	if *seg_a == 203 && *seg_b == 0 && *seg_c == 113 {
		*seg_c = 114
		return true
	}

	return false
}
