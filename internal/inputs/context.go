package inputs

import (
	"fmt"
	"os"
	"path/filepath"
)

func InputContext(config Config) error {
	repoName := filepath.Base(config.RepoPath)
	instructions := fmt.Sprintf(
		`

<context>
You are an expert software engineer who receives a summary of the repo "%s".
Think about the contents and purpose of the repo.
</context>
`, repoName)

	file, err := os.OpenFile(config.OutputFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(instructions); err != nil {
		return err
	}

	return nil
}
