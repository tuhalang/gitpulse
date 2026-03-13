package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/tuhalang/gitpulse/internal/github"
	"github.com/spf13/cobra"
)

var (
	countRepos string
	countDays  int
	countUser  string
)

var countCmd = &cobra.Command{
	Use:   "count",
	Short: "Count and list PRs for a user across repos",
	Long: `Count merged PRs for a user in specified repos within a time range.
Shows all commits and changes for each PR.

Example:
  gitpulse count --repos "owner/repo1,owner/repo2" --user tuhalang --days 5`,
	RunE: runCount,
}

func init() {
	rootCmd.AddCommand(countCmd)

	countCmd.Flags().StringVar(&countRepos, "repos", "", "Comma-separated list of repos (owner/repo format)")
	countCmd.Flags().IntVar(&countDays, "days", 0, "Number of days to look back (default: 7 or saved config)")
	countCmd.Flags().StringVar(&countUser, "user", "", "GitHub username (defaults to authenticated user or saved config)")
}

func runCount(cmd *cobra.Command, args []string) error {
	// Load config for defaults
	cfg, _ := loadConfig()

	// Apply config defaults if flags not provided
	if countRepos == "" && cfg.Repos != "" {
		countRepos = cfg.Repos
	}
	if countUser == "" && cfg.User != "" {
		countUser = cfg.User
	}
	if countDays == 0 {
		if cfg.Days > 0 {
			countDays = cfg.Days
		} else {
			countDays = 7
		}
	}

	// Validate repos
	if countRepos == "" {
		return fmt.Errorf("repos required: use --repos flag or 'gitpulse config set repos <value>'")
	}

	if err := github.CheckGhInstalled(); err != nil {
		return err
	}

	client := github.NewClient()

	// Get user
	username := countUser
	if username == "" {
		u, err := client.GetAuthenticatedUser()
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}
		username = u
	}

	// Parse repos
	repoNames := strings.Split(countRepos, ",")
	var repos []github.Repository
	for _, name := range repoNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		parts := strings.Split(name, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid repo format: %s (expected owner/repo)", name)
		}
		repos = append(repos, github.Repository{
			FullName: name,
			Owner:    parts[0],
			Name:     parts[1],
		})
	}

	if len(repos) == 0 {
		return fmt.Errorf("no valid repos provided")
	}

	// Calculate time range
	until := time.Now()
	since := until.AddDate(0, 0, -countDays)

	// Print header
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	bold := color.New(color.Bold)

	cyan.Printf("\n📊 PR Report: %s\n", username)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("   Repos: %s\n", countRepos)
	fmt.Printf("   Period: %s → %s (%d days)\n", since.Format("Jan 02"), until.Format("Jan 02, 2006"), countDays)
	fmt.Println(strings.Repeat("─", 60))

	// Fetch PRs
	fmt.Print("Fetching merged PRs...")
	prs, err := client.FetchMergedPRsFromRepos(repos, since, until, username)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("failed to fetch PRs: %w", err)
	}
	fmt.Printf(" found %d\n\n", len(prs))

	if len(prs) == 0 {
		fmt.Println("No merged PRs found in this period.")
		return nil
	}

	// Process each PR
	var totalAdditions, totalDeletions int

	for i, pr := range prs {
		repoName := pr.BaseBranch // We stored repo name here

		bold.Printf("\n%d. #%d %s\n", i+1, pr.Number, pr.Title)
		fmt.Printf("   Repo: %s\n", repoName)
		if pr.MergedAt != nil {
			fmt.Printf("   Merged: %s\n", pr.MergedAt.Format("Jan 02, 2006 15:04"))
		}
		fmt.Printf("   URL: %s\n", pr.URL)

		// Fetch commits for this PR
		parts := strings.Split(repoName, "/")
		if len(parts) == 2 {
			commits, err := client.GetPRCommits(parts[0], parts[1], pr.Number)
			if err == nil && len(commits) > 0 {
				fmt.Printf("\n   Commits (%d):\n", len(commits))

				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				var prAdditions, prDeletions int

				for _, c := range commits {
					prAdditions += c.Additions
					prDeletions += c.Deletions
					fmt.Fprintf(w, "     • %s\t%s\t%s\n",
						c.SHA[:7],
						fmt.Sprintf("%s / %s", green.Sprintf("+%d", c.Additions), red.Sprintf("-%d", c.Deletions)),
						truncate(c.Message, 40))
				}
				w.Flush()

				totalAdditions += prAdditions
				totalDeletions += prDeletions

				fmt.Printf("\n   PR Total: %s / %s\n",
					green.Sprintf("+%d", prAdditions),
					red.Sprintf("-%d", prDeletions))
			}
		}
	}

	// Summary
	fmt.Println()
	fmt.Println(strings.Repeat("═", 60))
	bold.Printf("📈 Summary\n")
	fmt.Printf("   Total PRs: %d\n", len(prs))
	fmt.Printf("   Total Changes: %s / %s\n",
		green.Sprintf("+%d", totalAdditions),
		red.Sprintf("-%d", totalDeletions))
	fmt.Println()

	return nil
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
