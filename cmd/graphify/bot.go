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
	cmd.AddCommand(newBotDoctorCmd())
	cmd.AddCommand(newBotPRReviewCmd())
	return cmd
}

// runBotScript execs a bots/*.py script, forwarding stdio and propagating
// its exit code, so `graphify bot X` behaves like running the script
// directly. Shared by every bot subcommand.
func runBotScript(scriptName string, botsDir string, pyArgs []string) error {
	bots, err := locateBotsDir(botsDir)
	if err != nil {
		return err
	}
	script := filepath.Join(bots, scriptName)
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("bot script not found at %s: %w", script, err)
	}
	python := exec.Command("python3", append([]string{script}, pyArgs...)...)
	python.Stdout = os.Stdout
	python.Stderr = os.Stderr
	python.Stdin = os.Stdin
	if err := python.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

func newBotDoctorCmd() *cobra.Command {
	var repoPath, botsDir, graphifyBin string
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Verify the orchestration chain: claude connects, gh authed, MCP server responds",
		Long: `Runs preflight checks for the whole bot orchestration:
python, the gh CLI (installed + authenticated), the claude CLI (installed +
able to actually get a completion), and graphify's own MCP server (does it
respond with its tools). Use this to confirm "our app can connect a Claude
session from the terminal" before running any bot.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoPath([]string{repoPath})
			if err != nil {
				return err
			}
			pyArgs := []string{"--repo", repoRoot}
			if graphifyBin != "" {
				pyArgs = append(pyArgs, "--graphify-bin", graphifyBin)
			}
			return runBotScript("preflight.py", botsDir, pyArgs)
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", ".", "path to the local repo checkout")
	cmd.Flags().StringVar(&botsDir, "bots-dir", "", "path to the bots/ directory (default: auto-detect)")
	cmd.Flags().StringVar(&graphifyBin, "graphify-bin", "", "path to the graphify binary (default: auto-detect)")
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
			pyArgs := []string{args[0], "--repo", repoRoot}
			if graphifyBin != "" {
				pyArgs = append(pyArgs, "--graphify-bin", graphifyBin)
			}
			if dryRun {
				pyArgs = append(pyArgs, "--dry-run")
			}
			return runBotScript("pr_review.py", botsDir, pyArgs)
		},
	}
	cmd.Flags().StringVar(&repoPath, "repo", ".", "path to the local repo checkout")
	cmd.Flags().StringVar(&botsDir, "bots-dir", "", "path to the bots/ directory (default: auto-detect)")
	cmd.Flags().StringVar(&graphifyBin, "graphify-bin", "", "path to the graphify binary (default: auto-detect)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the review instead of posting it as a PR comment")
	return cmd
}
