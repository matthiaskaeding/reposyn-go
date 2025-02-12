package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/plumbing/format/gitignore"
)

const (
	fileBufferSize     = 256 * 1024
	jobChannelBuffer   = 10000
	smallFileThreshold = 32 * 1024
)

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

type Config struct {
	inputDir       string
	outputFile     string
	textExtensions map[string]bool
	numWorkers     int
}

type FileJob struct {
	path    string
	relPath string
	size    int64
}

type BatchJob struct {
	files []FileJob
	size  int64
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

func worker(jobs <-chan interface{}, wg *sync.WaitGroup, writer *bufio.Writer, writerMutex *sync.Mutex) {
	defer wg.Done()

	// Create a larger buffer for each worker
	buf := make([]byte, fileBufferSize)

	// Create a write buffer for batching writes
	writeBuffer := strings.Builder{}
	writeBuffer.Grow(fileBufferSize * 2) // Pre-allocate space

	processFile := func(job FileJob) error {
		file, err := os.Open(job.path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Write file header to buffer
		writeBuffer.WriteString(fmt.Sprintf("\n<File = %v>\n", job.relPath))

		// Read and buffer file contents
		for {
			n, err := file.Read(buf)
			if n > 0 {
				writeBuffer.Write(buf[:n])
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
		}

		writeBuffer.WriteString(fmt.Sprintf("\n</File = %v>\n", job.relPath))
		return nil
	}

	for job := range jobs {
		switch j := job.(type) {
		case FileJob:
			if err := processFile(j); err != nil {
				fmt.Fprintf(os.Stderr, "Error processing file %s: %v\n", j.path, err)
			}
		case BatchJob:
			// Process batch of small files
			for _, f := range j.files {
				if err := processFile(f); err != nil {
					fmt.Fprintf(os.Stderr, "Error processing file %s: %v\n", f.path, err)
				}
			}
		}

		// If buffer is getting full, flush it
		if writeBuffer.Len() >= fileBufferSize {
			writerMutex.Lock()
			writer.WriteString(writeBuffer.String())
			writerMutex.Unlock()
			writeBuffer.Reset()
		}
	}

	// Flush remaining content
	if writeBuffer.Len() > 0 {
		writerMutex.Lock()
		writer.WriteString(writeBuffer.String())
		writerMutex.Unlock()
	}
}

func ConcatenateFiles(config Config) error {
	outFile, err := os.Create(config.outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer outFile.Close()

	// Use a larger buffer size for the writer
	writer := bufio.NewWriterSize(outFile, fileBufferSize*2)
	defer writer.Flush()

	matcher, err := LoadGitignore(config.inputDir)
	if err != nil {
		return err
	}

	// Create job channels
	jobs := make(chan interface{}, jobChannelBuffer)

	var writerMutex sync.Mutex
	var wg sync.WaitGroup

	// Start worker pool
	for i := 0; i < config.numWorkers; i++ {
		wg.Add(1)
		go worker(jobs, &wg, writer, &writerMutex)
	}

	// Collect small files for batching
	var currentBatch []FileJob
	var currentBatchSize int64

	// Walk directory and send jobs
	err = filepath.Walk(config.inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
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

		fileJob := FileJob{
			path:    path,
			relPath: relPath,
			size:    info.Size(),
		}

		// Batch small files together
		if info.Size() <= smallFileThreshold {
			currentBatch = append(currentBatch, fileJob)
			currentBatchSize += info.Size()

			// Send batch if it's full
			if currentBatchSize >= fileBufferSize {
				jobs <- BatchJob{files: currentBatch, size: currentBatchSize}
				currentBatch = make([]FileJob, 0, 100)
				currentBatchSize = 0
			}
		} else {
			// Send large files individually
			jobs <- fileJob
		}

		return nil
	})

	// Send remaining batch if any
	if len(currentBatch) > 0 {
		jobs <- BatchJob{files: currentBatch, size: currentBatchSize}
	}

	close(jobs)
	wg.Wait()

	return err
}

func main() {
	start := time.Now()
	var config Config

	flag.StringVar(&config.inputDir, "dir", "./repos/rust", "Directory to process")
	flag.StringVar(&config.outputFile, "out", "repo-synopsis.txt", "Output file path")
	numCPU := runtime.NumCPU()
	flag.IntVar(&config.numWorkers, "workers", numCPU, fmt.Sprintf("Number of worker goroutines (default: %d)", numCPU))
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
