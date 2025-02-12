package main // was 'reposyn' before

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"reposyn/internal/inputs" // needs to match your module name
)

func main() {
	start := time.Now()
	var config inputs.Config

	flag.StringVar(&config.InputDir, "dir", "./repos/rust", "Directory to process")
	flag.StringVar(&config.OutputFile, "out", "repo-synopsis.txt", "Output file path")
	numCPU := runtime.NumCPU()
	flag.IntVar(&config.NumWorkers, "workers", numCPU, fmt.Sprintf("Number of worker goroutines (default: %d)", numCPU))
	flag.Parse()

	config.TextExtensions = inputs.DefaultTextExtensions()

	fmt.Printf("Starting file concatenation with %d workers...\n", config.NumWorkers)

	if err := inputs.ConcatenateFiles(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Files successfully concatenated to %s\n", config.OutputFile)
	elapsed := time.Since(start)
	fmt.Printf("Operation took %s\n", elapsed)
}
