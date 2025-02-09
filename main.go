package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

type Config struct {
	inputDir       string
	outputFile     string
	textExtensions map[string]bool
	numWorkers     int
}

type FileJob struct {
	path    string
	relPath string
}

func DefaultTextExtensions() map[string]bool {
	return map[string]bool{
		".txt":  true,
		".md":   true,
		".go":   true,
		".json": true,
		".yaml": true,
		".yml":  true,
		".xml":  true,
		".html": true,
		".css":  true,
		".js":   true,
		".sh":   true,
		".conf": true,
		".toml": true,
	}
}

func LoadGitignore(repoPath string) (gitignore.Matcher, error) {
	patterns := make([]gitignore.Pattern, 0)
	gitignorePath := filepath.Join(repoPath, ".gitignore")

	if _, err := os.Stat(gitignorePath); err == nil {
		file, err := os.Open(gitignorePath)
		if err != nil {
			return nil, fmt.Errorf("error opening .gitignore: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				pattern := gitignore.ParsePattern(line, nil)
				patterns = append(patterns, pattern)
			}
		}
	}

	return gitignore.NewMatcher(patterns), nil
}

// worker processes files and writes directly to the output file
func worker(jobs <-chan FileJob, wg *sync.WaitGroup, writer *bufio.Writer, writerMutex *sync.Mutex) {
	defer wg.Done()

	// Create a buffer for each worker
	buf := make([]byte, 32*1024) // 32KB buffer

	for job := range jobs {
		// Open and read file
		file, err := os.Open(job.path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file %s: %v\n", job.path, err)
			continue
		}

		// Acquire lock and write file header
		writerMutex.Lock()
		fmt.Fprintf(writer, fmt.Sprintf("\n<File = %v>\n", job.relPath), job.relPath)
		writerMutex.Unlock()

		// Read and write file contents in chunks
		for {
			n, err := file.Read(buf)
			if n > 0 {
				writerMutex.Lock()
				writer.Write(buf[:n])
				writerMutex.Unlock()
			}
			if err != nil {
				break
			}
		}

		fmt.Fprintf(writer, fmt.Sprintf("\n</File = %v>\n", job.relPath), job.relPath)
		file.Close()
	}
}

func ConcatenateFiles(config Config) error {
	// Create output file
	outFile, err := os.Create(config.outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	defer writer.Flush()

	// Load .gitignore patterns
	matcher, err := LoadGitignore(config.inputDir)
	if err != nil {
		return err
	}

	// Create a buffered channel for jobs
	jobs := make(chan FileJob, 1000)

	// Create mutex for synchronized writing
	var writerMutex sync.Mutex

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < config.numWorkers; i++ {
		wg.Add(1)
		go worker(jobs, &wg, writer, &writerMutex)
	}

	// Walk directory and send jobs
	err = filepath.Walk(config.inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(config.inputDir, path)
		if err != nil {
			return fmt.Errorf("error getting relative path: %w", err)
		}

		if matcher.Match(strings.Split(relPath, string(os.PathSeparator)), false) {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !config.textExtensions[ext] {
			return nil
		}

		jobs <- FileJob{
			path:    path,
			relPath: relPath,
		}
		return nil
	})

	// Close jobs channel and wait for workers to finish
	close(jobs)
	wg.Wait()

	return err
}

func main() {
	start := time.Now()
	var config Config

	// Parse command line flags
	flag.StringVar(&config.inputDir, "dir", "./repos/ripgrep", "Directory to process")
	flag.StringVar(&config.outputFile, "out", "repo-synopsis.txt", "Output file path")
	numCPU := runtime.NumCPU()
	flag.IntVar(&config.numWorkers, "workers", numCPU, fmt.Sprintf("Number of worker goroutines (default: %d)", numCPU))
	flag.Parse()

	// Set up default text extensions
	config.textExtensions = DefaultTextExtensions()

	fmt.Printf("Starting file concatenation with %d workers...\n", config.numWorkers)

	// Process files
	if err := ConcatenateFiles(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Files successfully concatenated to %s\n", config.outputFile)
	elapsed := time.Since(start)
	fmt.Printf("Operation took %s\n", elapsed)

}
