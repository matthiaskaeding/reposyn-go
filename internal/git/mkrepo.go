package gitrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object" // Changed this import path to include v5	"github.com/go-git/go-git/v5"
)

func mkrepo() error {
	r, err := git.PlainInit("test_repos", false)
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
		"README.md":     "This is a test repository\nCreated for testing purposes\n",
		"example.txt":   "Some example content\nMultiple lines\nOf text\n",
		"data/info.txt": "Information in a subdirectory\n",
	}
	for i := 1; i < 100; i++ {
		fn := fmt.Sprintf("data/info_%v.txt", i)
		content := strings.Repeat("hello", i)
		files[fn] = content
	}

	for path, content := range files {
		dir := filepath.Dir(path)
		if dir != "." {
			fullDir := filepath.Join("test_repos", dir)
			if err := os.MkdirAll(fullDir, 0755); err != nil {
				return fmt.Errorf("error creating directory %s: %v", fullDir, err)
			}
		}

		fullPath := filepath.Join("test_repos", path)
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
			Name:  "Your Name",
			Email: "your@email.com",
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
