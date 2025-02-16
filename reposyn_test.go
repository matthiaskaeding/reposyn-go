package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBasics(t *testing.T) {
	summarizeRepo("./repos/dummy", "repo-synopsis.txt", false, "", "")
	contentByte, err := os.ReadFile("repo-synopsis.txt")
	if err != nil {
		log.Fatal(err)
	}
	content := string(contentByte)
	if !strings.Contains(string(content), "<File = README.md>") {
		t.Fatalf("README.md not included")
	}
	for i := 1; i < 9; i++ {
		baseName := fmt.Sprintf("info_%v.txt", i)
		fileName := filepath.Join("./repos/dummy/data", baseName)
		fileContent, err := os.ReadFile(fileName)
		if err != nil {
			log.Fatal(err)
		}
		fn := fmt.Sprintf("data/%v", baseName)
		desiredContent := fmt.Sprintf("<File = %v>\n%v\n</File = %v>",
			fn, string(fileContent), fn)
		if !strings.Contains(content, desiredContent) {
			msg := fmt.Sprintf("Needed %v", desiredContent)
			t.Fatalf("%s", msg)
		}
	}

	summarizeRepo("./repos/dummy", "repo-synopsis2.txt", false, "*.txt,*.json", "")
	fileContent, err := os.ReadFile("repo-synopsis2.txt")
	if err != nil {
		log.Fatal(err)
	}
	for i := 1; i < 9; i++ {
		ptrn := fmt.Sprintf("<File = info_%v.txt>", i)
		if strings.Contains(string(fileContent), ptrn) {
			t.Fatalf("Ignore not respected, %v in contents", ptrn)
		}
	}
	ptrn := "<File = config.json>"
	if strings.Contains(string(fileContent), ptrn) {
		t.Fatalf("Ignore not respected, %v in contents", ptrn)
	}

	t.Cleanup(func() {
		os.Remove("./repo-synopsis.txt")
		os.Remove("./repo-synopsis2.txt")
	})

}

func TestSummaryFeature(t *testing.T) {
	// Test summarizing text files
	err := summarizeRepo("./repos/dummy", "repo-synopsis-summary.txt", false, "", "*.txt")
	if err != nil {
		t.Fatalf("Failed to create repo summary: %v", err)
	}

	content, err := os.ReadFile("repo-synopsis-summary.txt")
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	contentStr := string(content)

	// Check that text files are summarized instead of included in full
	for i := 1; i < 9; i++ {
		fileName := fmt.Sprintf("data/info_%v.txt", i)

		// Verify summary format exists
		summaryStart := fmt.Sprintf("<Summary of file %v>", fileName)
		if !strings.Contains(contentStr, summaryStart) {
			t.Errorf("Missing summary for %v", fileName)
		}

		// Verify statistics section exists
		if !strings.Contains(contentStr, "<Statistics>") {
			t.Error("Missing statistics section in summary")
		}

		// Verify the file content is not included in full
		fullFileContent := strings.Repeat("hello\n", i)
		fullFileTag := fmt.Sprintf("<File = %v>\n%v</File = %v>",
			fileName, fullFileContent, fileName)
		if strings.Contains(contentStr, fullFileTag) {
			t.Errorf("File %v should be summarized but full content was found", fileName)
		}
	}

	// Verify non-txt files are still included in full
	if !strings.Contains(contentStr, "<File = README.md>") {
		t.Error("README.md should still be included in full")
	}
	if !strings.Contains(contentStr, "<File = config.json>") {
		t.Error("config.json should still be included in full")
	}

	t.Cleanup(func() {
		os.Remove("./repo-synopsis-summary.txt")
	})
}
