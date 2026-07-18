package main

import (
	"os"

	"github.com/spf13/cobra"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"graphify/internal/mcp"
)

func newMCPCmd() *cobra.Command {
	var dbPath string
	cmd := &cobra.Command{
		Use:   "mcp [path]",
		Short: "Parse a repo and run an MCP server over stdio for AI agents",
		Long: `Runs an MCP server speaking JSON-RPC over stdio, exposing the code graph
as tools (search_symbol, get_symbol, get_file_symbols, get_callers,
get_callees, get_file_slice, get_tree, get_stats) so an AI agent can query
the graph instead of reading whole files. Point an MCP-capable client at
"graphify mcp <path>" as the command to launch.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// stdio IS the MCP transport — nothing but the protocol may
			// touch stdout, so redirect all progress/log output to stderr.
			cmd.SetOut(os.Stderr)

			repoRoot, err := resolveRepoPath(args)
			if err != nil {
				return err
			}
			s, err := openStoreAndIndex(cmd, repoRoot, dbPath)
			if err != nil {
				return err
			}
			defer s.Close()

			srv := mcp.New(s, repoRoot)
			return mcpserver.ServeStdio(srv)
		},
	}
	cmd.Flags().StringVar(&dbPath, "db", "", "path to the SQLite graph DB (default: <repo>/.graphify/graph.db)")
	return cmd
}
