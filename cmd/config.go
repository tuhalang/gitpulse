package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Config holds saved default values
type Config struct {
	Repos string `json:"repos,omitempty"`
	User  string `json:"user,omitempty"`
	Days  int    `json:"days,omitempty"`
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gitpulse.json")
}

func loadConfig() (*Config, error) {
	path := getConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	path := getConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage saved configuration",
	Long:  `Save, remove, or clear default values for repos, user, and days.`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value (repos, user, days)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		switch key {
		case "repos":
			cfg.Repos = value
		case "user":
			cfg.User = value
		case "days":
			var days int
			if _, err := fmt.Sscanf(value, "%d", &days); err != nil {
				return fmt.Errorf("invalid days value: %s", value)
			}
			cfg.Days = days
		default:
			return fmt.Errorf("unknown key: %s (valid keys: repos, user, days)", key)
		}

		if err := saveConfig(cfg); err != nil {
			return err
		}

		fmt.Printf("✓ Set %s = %s\n", key, value)
		return nil
	},
}

var configAddCmd = &cobra.Command{
	Use:   "add repos <repo>",
	Short: "Add a repo to the list",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] != "repos" {
			return fmt.Errorf("can only add to 'repos' (got: %s)", args[0])
		}
		repo := args[1]

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		if cfg.Repos == "" {
			cfg.Repos = repo
		} else {
			cfg.Repos = cfg.Repos + "," + repo
		}

		if err := saveConfig(cfg); err != nil {
			return err
		}

		fmt.Printf("✓ Added %s\n", repo)
		fmt.Printf("  repos: %s\n", cfg.Repos)
		return nil
	},
}

var configRemoveCmd = &cobra.Command{
	Use:   "remove <key>",
	Short: "Remove a config value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		switch key {
		case "repos":
			cfg.Repos = ""
		case "user":
			cfg.User = ""
		case "days":
			cfg.Days = 0
		default:
			return fmt.Errorf("unknown key: %s", key)
		}

		if err := saveConfig(cfg); err != nil {
			return err
		}

		fmt.Printf("✓ Removed %s\n", key)
		return nil
	},
}

var configClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all saved config",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := getConfigPath()
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Println("✓ Config cleared")
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current config",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		fmt.Println("Current config:")
		if cfg.Repos != "" {
			fmt.Printf("  repos: %s\n", cfg.Repos)
		}
		if cfg.User != "" {
			fmt.Printf("  user:  %s\n", cfg.User)
		}
		if cfg.Days > 0 {
			fmt.Printf("  days:  %d\n", cfg.Days)
		}
		if cfg.Repos == "" && cfg.User == "" && cfg.Days == 0 {
			fmt.Println("  (empty)")
		}
		fmt.Printf("\nConfig file: %s\n", getConfigPath())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configAddCmd)
	configCmd.AddCommand(configRemoveCmd)
	configCmd.AddCommand(configClearCmd)
	configCmd.AddCommand(configShowCmd)
}
