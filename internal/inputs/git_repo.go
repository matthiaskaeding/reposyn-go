package inputs

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// FindGitRoot searches for a .git directory starting from the current directory
func FindGitRoot(startDir string) (string, error) {

	absPath, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		gitPath := filepath.Join(absPath, ".git")
		fileInfo, err := os.Stat(gitPath)
		if err == nil {
			if fileInfo.IsDir() {
				return absPath, nil
			}
		}

		// Move up to parent directory
		parent := filepath.Dir(absPath)
		if parent == absPath {
			return "", errors.New("no git repository found in current path or any parent directories")
		}

		absPath = parent
	}
}

func LoadGitignore(config Config) (gitignore.Matcher, error) {
	patterns := make([]gitignore.Pattern, 0)
	repoPath := config.RepoPath
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

	for _, s := range config.IgnorePatterns {
		pattern := gitignore.ParsePattern(s, nil)
		patterns = append(patterns, pattern)
	}

	return gitignore.NewMatcher(patterns), nil
}

func InputRepoStats(config Config) error {
	repo, err := git.PlainOpen(config.RepoPath)
	if err != nil {
		return fmt.Errorf("error opening repository: %w", err)
	}

	refs, err := repo.References()
	if err != nil {
		return fmt.Errorf("error getting references: %w", err)
	}

	var defaultRef *plumbing.Reference
	var ErrFoundHead = errors.New("found HEAD reference")

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().String() == "refs/remotes/origin/HEAD" {
			defaultRef = ref
			return ErrFoundHead
		}
		return nil
	})

	if defaultRef == nil {
		defaultRef, err = repo.Head()
		if err != nil {
			return fmt.Errorf("error getting HEAD: %w", err)
		}
	}

	logOptions := &git.LogOptions{
		From:  defaultRef.Hash(),
		Order: git.LogOrderCommitterTime,
		All:   false,
	}

	var builder strings.Builder
	n_commits := 0

	commitIter, err := repo.Log(logOptions)
	if err != nil {
		return err
	}
	defer commitIter.Close()

	builder.WriteString("<Repo statistics>\n")
	builder.WriteString("<Most recent commits, starting at most recent>\n")
	var ErrEnoughCommits = errors.New("reached commit limit")

	err = commitIter.ForEach(func(c *object.Commit) error {
		if n_commits >= 100 {
			return ErrEnoughCommits
		}
		if n_commits < 10 {
			builder.WriteString(fmt.Sprintf("<Commit message #%v>\n", n_commits))
			builder.WriteString(c.Message)
			builder.WriteString(fmt.Sprintf("</Commit message #%v>\n", n_commits))
		}
		n_commits += 1

		return nil
	})

	if err != nil && err != ErrEnoughCommits {
		return err
	}
	builder.WriteString("</Most recent commits, starting at most recent>\n")

	if err != ErrEnoughCommits {
		builder.WriteString(
			fmt.Sprintf(
				"<Number of commits>%v</Number of commits>\n",
				n_commits))
	} else {
		builder.WriteString("<Number of commits>> 100</Number of commits>")
	}
	builder.WriteString("</Repo statistics>\n")

	content := builder.String()
	// fmt.Print(content)
	file, err := os.OpenFile(config.OutputFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		return err
	}

	return nil
}

func MakeSummaryMatcher(config Config) (gitignore.Matcher, error) {
	patterns := make([]gitignore.Pattern, 0)
	for _, s := range config.SummaryPatterns {
		pattern := gitignore.ParsePattern(s, nil)
		patterns = append(patterns, pattern)
	}

	return gitignore.NewMatcher(patterns), nil
}
