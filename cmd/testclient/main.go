package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/praveenkullu/dsdemo/client"
)

func main() {
	vsAddr := flag.String("vs", "localhost:8000", "View service address (host:port)")
	flag.Parse()

	fmt.Printf("Starting test client\n")
	fmt.Printf("View Service at %s\n", *vsAddr)

	ck := client.MakeClient(*vsAddr)
	defer ck.Close()

	// Wait a bit for servers to be ready
	fmt.Println("\nWaiting for servers to be ready...")
	time.Sleep(2 * time.Second)

	// Test 1: Put and Get
	fmt.Println("\n=== Test 1: Basic Put and Get ===")
	fmt.Println("Put(a, 1)")
	ck.Put("a", "1")

	fmt.Println("Get(a)")
	value := ck.Get("a")
	fmt.Printf("Got value: %s\n", value)
	if value == "1" {
		fmt.Println("✓ Test 1 passed")
	} else {
		fmt.Println("✗ Test 1 failed")
	}

	// Test 2: Multiple puts
	fmt.Println("\n=== Test 2: Multiple Puts ===")
	fmt.Println("Put(b, 2)")
	ck.Put("b", "2")
	fmt.Println("Put(c, 3)")
	ck.Put("c", "3")

	fmt.Println("Get(b)")
	value = ck.Get("b")
	fmt.Printf("Got value: %s\n", value)

	fmt.Println("Get(c)")
	value = ck.Get("c")
	fmt.Printf("Got value: %s\n", value)

	if value == "3" {
		fmt.Println("✓ Test 2 passed")
	} else {
		fmt.Println("✗ Test 2 failed")
	}

	// Test 3: Get non-existent key
	fmt.Println("\n=== Test 3: Get Non-existent Key ===")
	fmt.Println("Get(nonexistent)")
	value = ck.Get("nonexistent")
	fmt.Printf("Got value: '%s'\n", value)
	if value == "" {
		fmt.Println("✓ Test 3 passed")
	} else {
		fmt.Println("✗ Test 3 failed")
	}

	// Test 4: Update existing key
	fmt.Println("\n=== Test 4: Update Existing Key ===")
	fmt.Println("Put(a, 100)")
	ck.Put("a", "100")

	fmt.Println("Get(a)")
	value = ck.Get("a")
	fmt.Printf("Got value: %s\n", value)
	if value == "100" {
		fmt.Println("✓ Test 4 passed")
	} else {
		fmt.Println("✗ Test 4 failed")
	}

	// Continuous operation mode
	fmt.Println("\n=== Continuous Operation Mode ===")
	fmt.Println("Performing continuous puts and gets...")
	fmt.Println("(Press Ctrl+C to stop)")

	counter := 0
	for {
		key := fmt.Sprintf("key%d", counter%10)
		value := fmt.Sprintf("value%d", counter)

		ck.Put(key, value)
		retrievedValue := ck.Get(key)

		if retrievedValue == value {
			fmt.Printf("✓ [%d] Put/Get %s=%s\n", counter, key, value)
		} else {
			fmt.Printf("✗ [%d] Mismatch! Expected %s, got %s\n", counter, value, retrievedValue)
		}

		counter++
		time.Sleep(1 * time.Second)
	}
}
