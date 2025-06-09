package cmd

import (
	"fmt"

	"github.com/shah1011/obscure/internal/config"
	"github.com/spf13/cobra"
)

var defaultOnly bool

var listProvidersCmd = &cobra.Command{
	Use:   "list-providers",
	Short: "List available cloud providers",
	Run: func(cmd *cobra.Command, args []string) {
		// Mapping internal keys to user-friendly names
		providers := map[string]string{
			"s3":  "Amazon S3",
			"gcs": "Google Cloud Storage",
			"b2":  "Backblaze B2",
		}

		// Get user's saved default provider key
		defaultProviderKey, err := config.GetUserDefaultProvider()
		if err != nil || defaultProviderKey == "" {
			fmt.Println("⚠️  No default provider is set.")
			return
		}

		// Get current session provider key
		currentProviderKey, err := config.GetSessionProvider()
		if err != nil {
			// If error getting current session provider, just proceed without it
			currentProviderKey = ""
		}

		// Show only the default provider (friendly name)
		if defaultOnly {
			name, exists := providers[defaultProviderKey]
			if !exists {
				fmt.Println("⚠️  Unknown provider key:", defaultProviderKey)
				return
			}
			fmt.Println("☁️", name)
			return
		}

		// Print all providers, appending (default), (current), or both
		for key, name := range providers {
			isDefault := key == defaultProviderKey
			isCurrent := key == currentProviderKey

			status := ""
			switch {
			case isDefault && isCurrent:
				status = " (default & current)"
			case isDefault:
				status = " (default)"
			case isCurrent:
				status = " (current)"
			}

			fmt.Printf("☁️  %s%s\n", name, status)
		}
	},
}

func init() {
	listProvidersCmd.Flags().BoolVar(&defaultOnly, "default", false, "Show only the default provider")
	rootCmd.AddCommand(listProvidersCmd)
}
