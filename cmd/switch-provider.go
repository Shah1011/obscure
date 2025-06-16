package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/shah1011/obscure/internal/config"
	"github.com/spf13/cobra"
)

var switchProviderCmd = &cobra.Command{
	Use:   "switch-provider",
	Short: "Switch the active cloud provider for this session",
	RunE: func(cmd *cobra.Command, args []string) error {
		providers := []string{"Amazon S3", "Google Cloud Storage", "Backblaze B2", "IDrive E2", "S3-compatible"}
		providerKeys := []string{"s3", "gcs", "b2", "idrive", "s3-compatible"}

		// Get flag value
		defaultFlag, err := cmd.Flags().GetString("default")
		if err != nil {
			return err
		}

		// Normalize input if provided
		defaultFlag = strings.ToLower(defaultFlag)

		var newProviderKey string
		var newProviderName string

		if defaultFlag != "" {
			// Validate flag value
			found := false
			for i, key := range providerKeys {
				if key == defaultFlag {
					newProviderKey = key
					newProviderName = providers[i]
					found = true
					break
				}
			}
			if !found {
				return errors.New("invalid --default provider value; allowed: s3, gcs, or b2")
			}
		} else {
			// No flag: prompt user to select provider interactively
			underline := "\033[4m"
			reset := "\033[0m"
			prompt := promptui.Select{
				Label: "Select active cloud provider for this session",
				Items: providers,
				Templates: &promptui.SelectTemplates{
					Label:    "{{ . }}",
					Active:   underline + "{{ . | green }}" + reset,
					Inactive: "{{ . }}",
					Selected: "☁️  Selected: {{ . | green }}",
				},
				Stdout: os.Stderr,
			}

			idx, _, err := prompt.Run()
			if err != nil {
				return fmt.Errorf("cloud provider selection cancelled or failed: %w", err)
			}

			newProviderKey = providerKeys[idx]
			newProviderName = providers[idx]
		}

		// Set session provider (active provider)
		if err := config.SetSessionProvider(newProviderKey); err != nil {
			return fmt.Errorf("failed to set session provider: %w", err)
		}

		// If --default flag was set, update default provider too
		if defaultFlag != "" {
			if err := config.SetUserDefaultProvider(newProviderKey); err != nil {
				return fmt.Errorf("failed to set default provider: %w", err)
			}
			fmt.Printf("✅ Default provider and session switched to %s\n", newProviderName)
		} else {
			fmt.Printf("✅ Session provider switched to %s\n", newProviderName)
		}

		return nil
	},
}

func init() {
	switchProviderCmd.Flags().String("default", "", "Also set this provider as default (s3, gcs, or b2)")
	rootCmd.AddCommand(switchProviderCmd)
}
