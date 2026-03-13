# GitPulse

A simple CLI tool to count merged PRs and show commit details for a GitHub user.

## Requirements

- Go 1.21+
- GitHub CLI (gh) installed and authenticated

## Installation

```bash
go build -o gitpulse .
```

## Usage

```bash
./gitpulse count --repos "owner/repo" --days 5 --user username
```

## Flags

- `--repos` - Comma-separated list of repos (required)
- `--user` - GitHub username (default: authenticated user)
- `--days` - Days to look back (default: 7)
