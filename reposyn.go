package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"reposyn/internal/inputs"
)

// Main function, create the repo summary and writes to destination
func summarizeRepo(repoPath string) (string, error) {
	start := time.Now()
	var config inputs.Config

	config.TextExtensions = inputs.DefaultTextExtensions()

	// TODO: make this argument for CLI later
	flag.StringVar(&config.InputDir, "dir", repoPath, "Directory to process")
	// TODO: make this argument for CLI later
	flag.StringVar(&config.OutputFile, "out", "repo-synopsis.txt", "Output file path")
	numCPU := runtime.NumCPU()
	flag.IntVar(&config.NumWorkers, "workers", numCPU, fmt.Sprintf("Number of worker goroutines (default: %d)", numCPU))
	flag.Parse()

	repoPath, err := inputs.FindGitRoot(config.InputDir)
	if err != nil {
		return "", fmt.Errorf("error finding repository: %w", err)
	}
	fmt.Printf("Found repo at %v\n", repoPath)

	flag.StringVar(&config.RepoPath, "RepoPath", repoPath, "Path of repo")

	if err := os.Remove("repo-synopsis.txt"); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to remove existing file: %v", err)
	}
	if _, err := os.Create("repo-synopsis.txt"); err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}

	inputs.InputRepoStats(config)

	fmt.Printf("Starting file concatenation with %d workers...\n", config.NumWorkers)
	if err := inputs.MergeFiles(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	dest := &config.OutputFile

	fmt.Printf("Files successfully concatenated to %s\n", config.OutputFile)

	inputs.InputContext(config)

	elapsed := time.Since(start)
	fmt.Printf("Operation took %s\n", elapsed)

	return *dest, nil
}

func main() {
	summarizeRepo("./")
}
