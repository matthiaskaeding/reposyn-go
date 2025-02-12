package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelloName(t *testing.T) {
	summarizeRepo("./repos/dummy")
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

}
