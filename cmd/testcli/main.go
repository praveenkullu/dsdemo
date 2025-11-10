package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/praveenkullu/dsdemo/client"
)

func main() {
	vsAddr := flag.String("vs", "localhost:8000", "View service address")
	operation := flag.String("op", "", "Operation: get, put, or view")
	key := flag.String("key", "", "Key for get/put operations")
	value := flag.String("value", "", "Value for put operation")
	flag.Parse()

	if *operation == "" {
		fmt.Println("Error: -op flag is required (get, put, or view)")
		os.Exit(1)
	}

	ck := client.MakeClient(*vsAddr)
	defer ck.Close()

	switch *operation {
	case "get":
		if *key == "" {
			fmt.Println("Error: -key flag is required for get operation")
			os.Exit(1)
		}
		result := ck.Get(*key)
		fmt.Printf("%s\n", result)

	case "put":
		if *key == "" || *value == "" {
			fmt.Println("Error: -key and -value flags are required for put operation")
			os.Exit(1)
		}
		ck.Put(*key, *value)
		fmt.Printf("OK\n")

	case "view":
		// Get view by calling GetView on the view service
		fmt.Println("View query not yet implemented in this simple client")
		os.Exit(1)

	default:
		fmt.Printf("Error: Unknown operation '%s'\n", *operation)
		os.Exit(1)
	}
}
