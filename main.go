package reposyn

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"internal/inputs"
)

const (
	fileBufferSize     = 256 * 1024
	jobChannelBuffer   = 10000
	smallFileThreshold = 32 * 1024
)

func main() {
	start := time.Now()
	var config inputs.Config

	flag.StringVar(&config.inputDir, "dir", "./repos/rust", "Directory to process")
	flag.StringVar(&inputs.config.outputFile, "out", "repo-synopsis.txt", "Output file path")
	numCPU := runtime.NumCPU()
	flag.IntVar(&inputs.config.numWorkers, "workers", numCPU, fmt.Sprintf("Number of worker goroutines (default: %d)", numCPU))
	flag.Parse()

	config.textExtensions = DefaultTextExtensions()

	fmt.Printf("Starting file concatenation with %d workers...\n", config.numWorkers)

	if err := ConcatenateFiles(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Files successfully concatenated to %s\n", config.outputFile)
	elapsed := time.Since(start)
	fmt.Printf("Operation took %s\n", elapsed)
}
