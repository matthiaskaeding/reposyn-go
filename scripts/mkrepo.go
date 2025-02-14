package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object" // Changed this import path to include v5	"github.com/go-git/go-git/v5"
)

func mkrepo() error {
	p := filepath.Join("repos", "dummy")
	if _, err := os.Stat(p); err == nil {
		if err := os.RemoveAll(p); err != nil {
			return fmt.Errorf("failed to remove existing directory: %v", err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking directory: %v", err)
	}

	err := os.MkdirAll(filepath.Join(p, "data"), 0755)
	if err != nil {
		fmt.Printf("Error making data folder: %v\n", err)
		return err
	}
	r, err := git.PlainInit(p, false)
	if err != nil {
		fmt.Printf("Error initializing repository: %v\n", err)
		return err
	}

	// Get the worktree
	w, err := r.Worktree()
	if err != nil {
		fmt.Printf("Error getting worktree: %v\n", err)
		return err
	}

	// Create some example files
	files := map[string]string{
		"README.md": "This is a test repository\nCreated for testing purposes\n",
	}
	for i := 1; i < 10; i++ {
		fn := filepath.Join("data", fmt.Sprintf("info_%v.txt", i))
		content := strings.Repeat("hello\n", i)
		files[fn] = content
	}

	for path, content := range files {
		fullPath := filepath.Join(p, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("error writing file %s: %v", path, err)
		}

		_, err = w.Add(path)
		if err != nil {
			return fmt.Errorf("error adding file %s: %v", path, err)
		}
	}

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Some Name",
			Email: "some@email.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		fmt.Printf("Error creating commit: %v\n", err)
		return err
	}

	if err != nil {
		fmt.Printf("Error creating commit: %v\n", err)
		return err
	}

	fmt.Printf("Repository created")
	return nil
}

func main() {
	err := mkrepo()
	if err != nil {
		log.Panic(err)
	}
}
