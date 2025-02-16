package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"reposyn/internal/inputs"

	"github.com/urfave/cli/v3"
	"golang.design/x/clipboard"
)

// Main function, create the repo summary and writes to destination
func summarizeRepo(targetDir string, outputFile string, wantClipboard bool) error {
	start := time.Now()

	repoPath, err := inputs.FindGitRoot(targetDir)
	if err != nil {
		return fmt.Errorf("error finding repository: %w", err)
	}
	fmt.Printf("Found repo at %v\n", repoPath)

	if wantClipboard {
		err = clipboard.Init()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while making clipboard %v\n", err)
			os.Exit(1)
		}
		tempFile, err := os.CreateTemp("", "prefix-*.txt")
		defer tempFile.Close()
		if err != nil {
			log.Fatal(err)
		}
		outputFile = tempFile.Name()
	} else {
		if err := os.Remove(outputFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing file: %v", err)
		}
		if _, err := os.Create(outputFile); err != nil {
			return fmt.Errorf("failed to create file: %v", err)
		}
	}

	config := inputs.Config{
		InputDir:       repoPath,
		OutputFile:     outputFile,
		TextExtensions: inputs.DefaultTextExtensions(),
		NumWorkers:     runtime.NumCPU(),
		RepoPath:       repoPath,
		Clipboard:      wantClipboard,
	}

	inputs.InputRepoStats(config)

	fmt.Printf("Starting file concatenation with %d workers...\n", config.NumWorkers)
	if err := inputs.MergeFiles(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	inputs.InputContext(config)

	elapsed := time.Since(start).Round(100 * time.Millisecond)
	if !wantClipboard {
		fmt.Printf("Files successfully concatenated to %s\n", config.OutputFile)
	} else {
		contents, err := os.ReadFile(outputFile)
		if err != nil {
			log.Fatal(err)
		}
		clipboard.Write(clipboard.FmtText, contents)
		fmt.Printf("Files successfully concatenated to clipboard\n")
	}
	fmt.Printf("Operation took %s\n", elapsed)

	return nil
}

func main() {
	app := &cli.Command{
		Name:  "reposyn",
		Usage: "Create AI friendly repo summary",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "target",
				Aliases: []string{"t"},
				Value:   "./",
				Usage:   "Target directory path",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "repo-synopsis.txt",
				Usage:   "Output text file",
			},
			&cli.BoolFlag{
				Name:    "clipboard",
				Aliases: []string{"c"},
				Value:   false,
				Usage:   "Write output to clipboard",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			target := c.String("target")
			output := c.String("output")
			clipboard := c.Bool("clipboard")

			err := summarizeRepo(target, output, clipboard)
			if err != nil {
				return fmt.Errorf("failed to summarize repo: %w", err)
			}

			return nil
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
