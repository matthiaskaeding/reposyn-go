package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"reposyn/internal/files" // needs to match your module name
)

func summarizeRepo(repoPath string) {
	start := time.Now()
	var config files.Config

	config.TextExtensions = files.DefaultTextExtensions()
	flag.StringVar(&config.InputDir, "dir", repoPath, "Directory to process")
	flag.StringVar(&config.OutputFile, "out", "repo-synopsis.txt", "Output file path")
	numCPU := runtime.NumCPU()
	flag.IntVar(&config.NumWorkers, "workers", numCPU, fmt.Sprintf("Number of worker goroutines (default: %d)", numCPU))
	flag.Parse()
	if err := files.MergeFiles(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Starting file concatenation with %d workers...\n", config.NumWorkers)

	fmt.Printf("Files successfully concatenated to %s\n", config.OutputFile)
	elapsed := time.Since(start)
	fmt.Printf("Operation took %s\n", elapsed)
}

func main() {
	summarizeRepo("./repos/rust")
}
