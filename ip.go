package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Parse an IP string into its segments.
func parseIP(ip string) (int, int, int, int, error) {
	var segA, segB, segC, segD int

	if ip == "" {
		return 1, 0, 0, 0, nil
	}

	octets := strings.Split(ip, ".")
	if len(octets) != 4 {
		return 0, 0, 0, 0, fmt.Errorf("invalid IP address '%s'", ip)
	}

	segA, err := strconv.Atoi(octets[0])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("atoi failed on '%s', %s", ip, err)
	}

	segB, err = strconv.Atoi(octets[1])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("atoi failed on '%s', %s", ip, err)
	}

	segC, err = strconv.Atoi(octets[2])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("atoi failed on '%s', %s", ip, err)
	}

	segD, err = strconv.Atoi(octets[3])
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("atoi failed on '%s', %s", ip, err)
	}

	if segA > 255 || segB > 255 || segC > 255 || segD > 255 {
		return 0, 0, 0, 0, fmt.Errorf("invalid IP address '%s'", ip)
	}

	if isReserved(&segA, &segB, &segC) {
		fmt.Println("IP is reserved, skipping to next available range")
	}

	fmt.Printf("Starting from IP '%d.%d.%d.%d'\n", segA, segB, segC, segD)

	return segA, segB, segC, segD, nil
}

// Check if the IP is in any reserved range, skips to the next available range if it is.
func isReserved(segA *int, segB *int, segC *int) bool {
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
