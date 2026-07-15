package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a default config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path := filepath.Join(home, ".goscrape.yaml")
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config file already exists at %s", path)
		}

		content := `# GoScrape Configuration
default_output: ~/goscrape-output
default_delay: 1s
default_workers: 3
default_format: json
default_depth: 2
default_max_pages: 100
default_timeout: 30s
default_retries: 2
rotate_agents: true
user_agent: GoScrape/1.0
`

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
		fmt.Printf("Config file created at %s\n", path)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("default_output: %s\n", GetDefaultOutputDir())
		fmt.Printf("default_delay: %s\n", GetConfigString("default_delay"))
		fmt.Printf("default_workers: %d\n", GetConfigInt("default_workers"))
		fmt.Printf("default_format: %s\n", GetConfigString("default_format"))
		fmt.Printf("default_depth: %d\n", GetConfigInt("default_depth"))
		fmt.Printf("default_max_pages: %d\n", GetConfigInt("default_max_pages"))
		fmt.Printf("default_timeout: %s\n", GetConfigString("default_timeout"))
		fmt.Printf("default_retries: %d\n", GetConfigInt("default_retries"))
		fmt.Printf("rotate_agents: %t\n", GetConfigBool("rotate_agents"))
		fmt.Printf("user_agent: %s\n", GetConfigString("user_agent"))
	},
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}