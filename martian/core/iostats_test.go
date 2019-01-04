// Copyright (c) 2018 10X Genomics, Inc. All rights reserved.

package core

import (
	"fmt"
	"time"
)

func ExampleIoStatsBuilder() {
	// Set the start time to a known value.
	t := time.Now()
	sb := NewIoStatsBuilder()
	sb.lastMeasurement = t
	sb.start = t

	// No writes.
	// Reads at a constant rate of 2048 bytes every 10 seconds, with
	// 11, 9, and 10 syscalls in the first, second, and third
	// 10-second period, respectively.
	sb.Update(map[int]*IoAmount{
		1: {
			Read:  IoValues{Syscalls: 1, BlockBytes: 1024},
			Write: IoValues{},
		},
		2: {
			Read:  IoValues{Syscalls: 10, BlockBytes: 1024},
			Write: IoValues{},
		},
	}, t.Add(time.Second*10))
	sb.Update(map[int]*IoAmount{
		1: {
			Read:  IoValues{Syscalls: 10, BlockBytes: 3072},
			Write: IoValues{},
		},
		2: {
			Read:  IoValues{Syscalls: 10, BlockBytes: 1024},
			Write: IoValues{},
		},
	}, t.Add(time.Second*20))
	sb.Update(map[int]*IoAmount{
		1: {
			Read:  IoValues{Syscalls: 10, BlockBytes: 4096},
			Write: IoValues{},
		},
		3: {
			Read:  IoValues{Syscalls: 10, BlockBytes: 1024},
			Write: IoValues{},
		},
	}, t.Add(time.Second*30))
	fmt.Println("Read syscalls:")
	fmt.Println("Total:", sb.Total.Read.Syscalls)
	fmt.Printf("Rate: %0.1f ± %0.2f (max: %0.1f)\n\n",
		float64(sb.Total.Read.Syscalls)/30,
		sb.RateDev.Read.Syscalls,
		sb.RateMax.Read.Syscalls)
	fmt.Println("Write syscalls:")
	fmt.Println("Total:", sb.Total.Write.Syscalls)
	fmt.Printf("Rate: %0.1f ± %0.2f (max: %0.1f)\n\n",
		float64(sb.Total.Write.Syscalls)/30,
		sb.RateDev.Write.Syscalls,
		sb.RateMax.Write.Syscalls)
	fmt.Println("Read bytes:")
	fmt.Println("Total:", sb.Total.Read.BlockBytes)
	fmt.Printf("Rate: %0.1f ± %0.2f (max: %0.1f)\n\n",
		float64(sb.Total.Read.BlockBytes)/30,
		sb.RateDev.Read.BlockBytes,
		sb.RateMax.Read.BlockBytes)
	fmt.Println("Write bytes:")
	fmt.Println("Total:", sb.Total.Write.BlockBytes)
	fmt.Printf("Rate: %0.1f ± %0.2f (max: %0.1f)\n",
		float64(sb.Total.Write.BlockBytes)/30,
		sb.RateDev.Write.BlockBytes,
		sb.RateMax.Write.BlockBytes)
	// Output:
	// Read syscalls:
	// Total: 30
	// Rate: 1.0 ± 0.08 (max: 1.1)
	//
	// Write syscalls:
	// Total: 0
	// Rate: 0.0 ± 0.00 (max: 0.0)
	//
	// Read bytes:
	// Total: 6144
	// Rate: 204.8 ± 0.00 (max: 204.8)
	//
	// Write bytes:
	// Total: 0
	// Rate: 0.0 ± 0.00 (max: 0.0)
}
