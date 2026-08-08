package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

// locateBotsDir finds the bots/ directory the same way locateWebDir finds
// web/dist: next to the installed binary, in the current working
// directory, or in this source tree during local development.
func locateBotsDir(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	var candidates []string
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "bots"))
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "bots"))
	}
	if _, thisFile, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(thisFile))), "bots"))
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("bots/ directory not found (looked next to the binary, in cwd, and in the graphify source tree) — pass --bots-dir")
}

func newBotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bot",
		Short: "Run an AI bot (see bots/README.md for the full list)",
	}
	cmd.AddCommand(newBotPRReviewCmd())
	return cmd
}

func newBotPRReviewCmd() *cobra.Command {
	var repoPath, botsDir, graphifyBin string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "pr-review <pr-number>",
		Short: "Review an open PR's diff and post findings as a comment",
		Long: `Reviews an open GitHub PR's diff with Claude, using graphify's own MCP
server as the model's source of repo context (search the code graph,
don't read whole files), then posts the findings as a PR comment.

Requires: gh (authenticated), claude (authenticated — a local
`+"`claude login`"+` session, or ANTHROPIC_API_KEY in CI), and a graphify
binary (this one, or one found on PATH).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoPath([]string{repoPath})
			if err != nil {
				return err
			}
			bots, err := locateBotsDir(botsDir)
			if err != nil {
				return err
			}
			script := filepath.Join(bots, "pr_review.py")
			if _, err := os.Stat(script); err != nil {
				return fmt.Errorf("bot script not found at %s: %w", script, err)
			}

			pyArgs := []string{script, args[0], "--repo", repoRoot}
			if graphifyBin != "" {
				pyArgs = append(pyArgs, "--graphify-bin", graphifyBin)
			}
			if dryRun {
				pyArgs = append(pyArgs, "--dry-run")
			}

			python := exec.Command("python3", pyArgs...)
			python.Stdout = os.Stdout
			python.Stderr = os.Stderr
			python.Stdin = os.Stdin
			if err := python.Run(); err != nil {
				// The bot script already prints its own error to stderr —
				// exit with its code directly instead of cobra also
				// printing a redundant "graphify: exit status N".
				if exitErr, ok := err.(*exec.ExitError); ok {
					os.Exit(exitErr.ExitCode())
				}
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", ".", "path to the local repo checkout")
	cmd.Flags().StringVar(&botsDir, "bots-dir", "", "path to the bots/ directory (default: auto-detect)")
	cmd.Flags().StringVar(&graphifyBin, "graphify-bin", "", "path to the graphify binary (default: auto-detect)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the review instead of posting it as a PR comment")
	return cmd
}
