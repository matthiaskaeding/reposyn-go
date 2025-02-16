package inputs

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type FileSummary struct {
	TotalLines   int
	FirstThree   []string
	LastThree    []string
	EmptyLines   int
	AverageBytes float64
}

func SummarizeFile(filepath string) (*FileSummary, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	totalBytes := 0
	emptyLines := 0

	// Read all lines
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		totalBytes += len(line)
		if len(strings.TrimSpace(line)) == 0 {
			emptyLines++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	totalLines := len(lines)
	if totalLines == 0 {
		return nil, fmt.Errorf("file is empty")
	}

	// Calculate average bytes per line
	averageBytes := float64(totalBytes) / float64(totalLines)

	// Get first and last three lines
	firstThree := make([]string, 0, 3)
	lastThree := make([]string, 0, 3)

	for i := 0; i < min(3, totalLines); i++ {
		firstThree = append(firstThree, lines[i])
	}

	for i := max(0, totalLines-3); i < totalLines; i++ {
		lastThree = append(lastThree, lines[i])
	}

	summary := &FileSummary{
		TotalLines:   totalLines,
		FirstThree:   firstThree,
		LastThree:    lastThree,
		EmptyLines:   emptyLines,
		AverageBytes: averageBytes,
	}

	return summary, nil
}

func WriteFileSummary(filepath string, relPath string, builder *strings.Builder) error {
	summary, err := SummarizeFile(filepath)
	if err != nil {
		return err
	}

	builder.WriteString(fmt.Sprintf("<Summary of file %v>\n", relPath))

	builder.WriteString("<First three lines>\n")
	for i, line := range summary.FirstThree {
		fmt.Fprintf(builder, "<line index=\"%d\"><%s></line>\n", i+1, line)
	}
	builder.WriteString("</First three lines>\n")

	builder.WriteString("<Last three lines>\n")
	for i, line := range summary.LastThree {
		fmt.Fprintf(builder, "<line index=\"%d\"><%s></line>\n", i+1, line)
	}
	builder.WriteString("</Last three lines>\n")

	builder.WriteString("<Statistics>\n")
	fmt.Fprintf(builder, "<Total lines>%d</Total lines>\n", summary.TotalLines)
	fmt.Fprintf(builder, "<Empty lines>%d</Empty lines>\n", summary.EmptyLines)
	fmt.Fprintf(builder, "<Average bytest per line>%.2f</Average bytest per line>\n", summary.AverageBytes)
	builder.WriteString("</Statistics>\n")

	builder.WriteString(fmt.Sprintf("</Summary of file %v>\n", relPath))
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
