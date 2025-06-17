package cmd

import (
	"fmt"

	"github.com/shah1011/obscure/internal/config"
	"github.com/spf13/cobra"
)

var whichProviderCmd = &cobra.Command{
	Use:   "which-provider",
	Short: "Prints the currently selected cloud provider in one line",
	Run: func(cmd *cobra.Command, args []string) {
		// Mapping internal keys to user-friendly names
		providers := map[string]string{
			"s3":            "Amazon S3",
			"gcs":           "Google Cloud Storage",
			"b2":            "Backblaze B2",
			"idrive":        "IDrive E2",
			"s3-compatible": "S3-compatible",
			"storj":         "Storj",
		}

		// Get current session provider
		currentProviderKey, err := config.GetSessionProvider()
		if err != nil || currentProviderKey == "" {
			// Fall back to default provider if session provider isn't set
			currentProviderKey, err = config.GetUserDefaultProvider()
			if err != nil || currentProviderKey == "" {
				fmt.Println("No cloud provider is selected.")
				return
			}
		}

		name, exists := providers[currentProviderKey]
		if !exists {
			// For s3-compatible providers, try to get the custom name
			if currentProviderKey == "s3-compatible" {
				providerConfig, err := config.GetProviderConfig("s3-compatible")
				if err == nil && providerConfig.CustomName != "" {
					fmt.Printf("☁️  %s (S3-compatible)\n", providerConfig.CustomName)
					return
				}
			}
			fmt.Println("Unknown provider key:", currentProviderKey)
			return
		}

		fmt.Println("☁️ ", name)
	},
}

func init() {
	rootCmd.AddCommand(whichProviderCmd)
}
