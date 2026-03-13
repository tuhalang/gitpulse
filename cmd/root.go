package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gitpulse",
	Short: "GitPulse - Count and analyze your GitHub PRs",
	Long: `GitPulse is a CLI tool that counts merged PRs and shows commit details.

Example:
  gitpulse count --repos "owner/repo1,owner/repo2" --days 7

Requires gh CLI to be installed and authenticated (run 'gh auth login').`,
}

func Execute() error {
	return rootCmd.Execute()
}
