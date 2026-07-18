package main

import (
	"fmt"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"graphify/internal/server"
)

func newServeCmd() *cobra.Command {
	var dbPath, webDir string
	var port int
	var open bool
	cmd := &cobra.Command{
		Use:   "serve [path]",
		Short: "Parse a repo and serve the graph API + web visualizer",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := resolveRepoPath(args)
			if err != nil {
				return err
			}
			s, err := openStoreAndIndex(cmd, repoRoot, dbPath)
			if err != nil {
				return err
			}
			defer s.Close()

			resolvedWebDir := locateWebDir(webDir)
			if resolvedWebDir == "" {
				cmd.Println("No built web UI found (looked for web/dist). Serving the API only — see API_CONTRACT.md, or pass --web-dir.")
			}

			srv := server.New(s, repoRoot, resolvedWebDir)
			addr := fmt.Sprintf("localhost:%d", port)
			url := "http://" + addr

			cmd.Printf("graphify serving %s\n", url)
			if open {
				openBrowser(url)
			}
			return http.ListenAndServe(addr, srv)
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "path to the SQLite graph DB (default: <repo>/.graphify/graph.db)")
	cmd.Flags().StringVar(&webDir, "web-dir", "", "path to the built web UI (web/dist); auto-detected if omitted")
	cmd.Flags().IntVar(&port, "port", 8420, "port to listen on")
	cmd.Flags().BoolVar(&open, "open", true, "open the web UI in a browser on start")
	return cmd
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
