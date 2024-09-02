package main

import (
	"flag"
	"fmt"
)

const (
	DEM_IsCompressed  = 0x40
	DEMO_BUFFER_SIZE  = 2 * 1024 * 1024
	VALVE_HEADER_SIZE = 16
)

func main() {
	filePath := flag.String("f", "", "Path to the demo file")
	outputToFile := flag.Bool("o", false, "Output to file")
	flag.Parse()

	if *filePath == "" {
		fmt.Println("Please provide a path to the demo file using the -f flag.")
		return
	}

	parser, err := NewDemoParser(*filePath, *outputToFile)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer parser.Close()

	if err := parser.Parse(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
