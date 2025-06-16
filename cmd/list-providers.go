package cmd

import (
	"fmt"
	"strings"

	"github.com/shah1011/obscure/internal/config"
	"github.com/spf13/cobra"
)

var defaultOnly bool

var listProvidersCmd = &cobra.Command{
	Use:   "list-providers",
	Short: "List configured cloud providers",
	Run: func(cmd *cobra.Command, args []string) {
		// Load configured providers
		providers, err := config.LoadUserProviders()
		if err != nil {
			fmt.Printf("❌ Failed to load providers: %v\n", err)
			return
		}

		if len(providers.Providers) == 0 {
			fmt.Println("📦 No providers configured")
			fmt.Println("   Run './obscure provider add <provider>' to add a provider")
			return
		}

		// Get user's saved default provider key
		defaultProviderKey, err := config.GetUserDefaultProvider()
		if err != nil {
			defaultProviderKey = ""
		}

		// Get current session provider key
		currentProviderKey, err := config.GetSessionProvider()
		if err != nil {
			currentProviderKey = ""
		}

		fmt.Println("📦 Configured cloud providers:")
		fmt.Println()

		for providerKey, providerConfig := range providers.Providers {
			// Check if configuration is complete
			isComplete, missing := config.IsProviderConfigComplete(providerConfig)

			// Determine provider display name
			var displayName string
			switch providerKey {
			case "s3":
				displayName = "Amazon S3"
			case "gcs":
				displayName = "Google Cloud Storage"
			case "b2":
				displayName = "Backblaze B2"
			case "idrive":
				displayName = "IDrive E2"
			case "s3-compatible":
				if providerConfig.CustomName != "" {
					displayName = fmt.Sprintf("%s (S3-compatible)", providerConfig.CustomName)
				} else {
					displayName = "S3-compatible"
				}
			default:
				displayName = strings.ToUpper(providerKey)
			}

			// Determine status
			var status string
			if !providerConfig.Enabled {
				status = "❌ Disabled"
			} else if !isComplete {
				status = "⚠️  Incomplete"
			} else {
				status = "✅ Enabled"
			}

			// Add default/current indicators
			if providerKey == defaultProviderKey {
				status += " (default)"
			}
			if providerKey == currentProviderKey {
				status += " (current)"
			}

			fmt.Printf("☁️  %s: %s\n", displayName, status)

			if !isComplete {
				fmt.Printf("    Missing: %s\n", strings.Join(missing, ", "))
			} else {
				// Show provider-specific details
				switch providerKey {
				case "s3":
					fmt.Printf("    Bucket: %s\n", providerConfig.Bucket)
					fmt.Printf("    Region: %s\n", providerConfig.Region)
				case "gcs":
					fmt.Printf("    Project: %s\n", providerConfig.ProjectID)
					fmt.Printf("    Service Account: %s\n", providerConfig.ServiceAccount)
				case "b2":
					fmt.Printf("    Bucket: %s\n", providerConfig.Bucket)
					fmt.Printf("    Endpoint: %s\n", providerConfig.Endpoint)
				case "idrive":
					fmt.Printf("    Bucket: %s\n", providerConfig.Bucket)
					fmt.Printf("    Region: %s\n", providerConfig.Region)
					fmt.Printf("    Endpoint: %s\n", providerConfig.IDriveEndpoint)
				case "s3-compatible":
					fmt.Printf("    Bucket: %s\n", providerConfig.Bucket)
					fmt.Printf("    Region: %s\n", providerConfig.Region)
					fmt.Printf("    Endpoint: %s\n", providerConfig.S3CompatibleEndpoint)
				}
			}
			fmt.Println()
		}

		// Show available provider types
		fmt.Println("📋 Available provider types:")
		fmt.Println("   • s3 - Amazon S3")
		fmt.Println("   • gcs - Google Cloud Storage")
		fmt.Println("   • b2 - Backblaze B2")
		fmt.Println("   • idrive - IDrive E2")
		fmt.Println("   • s3-compatible - Any S3-compatible service (Wasabi, DigitalOcean, etc.)")
		fmt.Println()
		fmt.Println("💡 Use './obscure provider add <type>' to add a new provider")
	},
}

func init() {
	listProvidersCmd.Flags().BoolVar(&defaultOnly, "default", false, "Show only the default provider")
	rootCmd.AddCommand(listProvidersCmd)
}
