package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"reposyn/internal/inputs" // needs to match your module name
)

// Main function, create the repo summary and writes to destination
func summarizeRepo(repoPath string) string {
	start := time.Now()
	var config inputs.Config

	config.TextExtensions = inputs.DefaultTextExtensions()
	flag.StringVar(&config.InputDir, "dir", repoPath, "Directory to process")
	flag.StringVar(&config.OutputFile, "out", "repo-synopsis.txt", "Output file path")
	numCPU := runtime.NumCPU()
	flag.IntVar(&config.NumWorkers, "workers", numCPU, fmt.Sprintf("Number of worker goroutines (default: %d)", numCPU))
	flag.Parse()

	fmt.Printf("Starting file concatenation with %d workers...\n", config.NumWorkers)
	if err := inputs.MergeFiles(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	dest := &config.OutputFile

	fmt.Printf("Files successfully concatenated to %s\n", config.OutputFile)
	elapsed := time.Since(start)
	fmt.Printf("Operation took %s\n", elapsed)

	return *dest
}

func main() {
	summarizeRepo("./repos/rust")
}
