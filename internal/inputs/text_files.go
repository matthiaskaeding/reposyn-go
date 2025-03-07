package inputs

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	InputDir        string
	OutputFile      string
	TextExtensions  map[string]bool
	NumWorkers      int
	RepoPath        string
	Clipboard       bool
	IgnorePatterns  []string
	SummaryPatterns []string
}

type FileJob struct {
	path      string
	relPath   string
	size      int64
	summarize bool
}

type BatchJob struct {
	files []FileJob
	size  int64
}

func worker(jobs <-chan interface{}, wg *sync.WaitGroup, writer *bufio.Writer, writerMutex *sync.Mutex) {
	defer wg.Done()

	fileBuffer := make([]byte, fileBufferSize)
	stringBuffer := strings.Builder{}
	stringBuffer.Grow(fileBufferSize * 2) // Pre-allocate space

	processFile := func(job FileJob) error {
		file, err := os.Open(job.path)
		if err != nil {
			return err
		}
		defer file.Close()
		if !job.summarize {
			// Just write whole file to buffer
			stringBuffer.WriteString(fmt.Sprintf("\n<File = %v>\n", job.relPath))
			for {
				n, err := file.Read(fileBuffer)
				if n > 0 {
					stringBuffer.Write(fileBuffer[:n])
				}
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}
			}
			stringBuffer.WriteString(fmt.Sprintf("\n</File = %v>\n", job.relPath))
		} else {
			WriteFileSummary(job.path, job.relPath, &stringBuffer)
		}
		return nil
	}

	for job := range jobs {
		switch j := job.(type) {
		case FileJob:
			if err := processFile(j); err != nil {
				fmt.Fprintf(os.Stderr, "Error processing file %s: %v\n", j.path, err)
			}
		case BatchJob:
			for _, f := range j.files {
				if err := processFile(f); err != nil {
					fmt.Fprintf(os.Stderr, "Error processing file %s: %v\n", f.path, err)
				}
			}
		}

		// Flush buffer if full
		if stringBuffer.Len() >= fileBufferSize {
			writerMutex.Lock()
			writer.WriteString(stringBuffer.String())
			writerMutex.Unlock()
			stringBuffer.Reset()
		}
	}

	// Flush remaining content
	if stringBuffer.Len() > 0 {
		writerMutex.Lock()
		writer.WriteString(stringBuffer.String())
		writerMutex.Unlock()
	}
}

func MergeFiles(config Config) error {
	outFile, err := os.OpenFile(config.OutputFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening output file: %w", err)
	}
	defer outFile.Close()

	writer := bufio.NewWriterSize(outFile, fileBufferSize*2)
	defer writer.Flush()

	matcher, err := LoadGitignore(config)
	if err != nil {
		return err
	}

	summaryMatcher, err := MakeSummaryMatcher(config)
	if err != nil {
		return err
	}

	jobs := make(chan interface{}, jobChannelBuffer)

	var writerMutex sync.Mutex
	var wg sync.WaitGroup

	// Start worker fill
	for i := 0; i < config.NumWorkers; i++ {
		wg.Add(1)
		go worker(jobs, &wg, writer, &writerMutex)
	}

	// Collect small files for batching
	var currentBatch []FileJob
	var currentBatchSize int64

	writer.WriteString("\n<Files>\n")

	// Walk directory and send jobs
	err = filepath.Walk(config.InputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		relPath, err := filepath.Rel(config.InputDir, path)
		if err != nil {
			return fmt.Errorf("error getting relative path: %w", err)
		}

		if matcher.Match(strings.Split(relPath, string(os.PathSeparator)), false) {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !config.TextExtensions[ext] {
			return nil
		}

		shouldBeSummarized := summaryMatcher.Match(strings.Split(relPath, string(os.PathSeparator)), false)

		fileJob := FileJob{
			path:      path,
			relPath:   relPath,
			size:      info.Size(),
			summarize: shouldBeSummarized,
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

	writer.WriteString("\n</Files>")

	return err
}
