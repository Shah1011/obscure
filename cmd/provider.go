package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	cfg "github.com/shah1011/obscure/internal/config"
	"github.com/spf13/cobra"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage cloud storage providers",
	Long: `Manage your cloud storage providers. You can add, remove, and configure
your own cloud storage accounts for backup storage.`,
}

var addProviderCmd = &cobra.Command{
	Use:   "add [s3|gcs]",
	Short: "Add a new cloud storage provider",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider := strings.ToLower(args[0])
		if provider != "s3" && provider != "gcs" {
			fmt.Println("❌ Invalid provider. Use 's3' or 'gcs'")
			return
		}

		config := &cfg.CloudProviderConfig{
			Provider: provider,
			Enabled:  true,
		}

		switch provider {
		case "s3":
			// Prompt for S3 credentials
			bucket, err := promptLine("Enter S3 bucket name: ")
			if err != nil {
				fmt.Println("❌ Invalid bucket name")
				return
			}
			region, err := promptLine("Enter AWS region (e.g., us-east-1): ")
			if err != nil {
				fmt.Println("❌ Invalid region")
				return
			}
			accessKey, err := promptLine("Enter AWS Access Key ID: ")
			if err != nil {
				fmt.Println("❌ Invalid access key")
				return
			}
			secretKey, err := promptPassword("Enter AWS Secret Access Key: ")
			if err != nil {
				fmt.Println("❌ Invalid secret key")
				return
			}

			config.Bucket = bucket
			config.Region = region
			config.AccessKeyID = accessKey
			config.SecretAccessKey = secretKey

		case "gcs":
			// Prompt for GCS credentials
			projectID, err := promptLine("Enter Google Cloud Project ID: ")
			if err != nil {
				fmt.Println("❌ Invalid project ID")
				return
			}
			serviceAccountPath, err := promptLine("Enter path to service account key file: ")
			if err != nil {
				fmt.Println("❌ Invalid service account path")
				return
			}

			// Verify service account file exists
			if _, err := os.Stat(serviceAccountPath); os.IsNotExist(err) {
				fmt.Println("❌ Service account file not found")
				return
			}

			config.ProjectID = projectID
			config.ServiceAccount = serviceAccountPath
		}

		// Save the configuration
		if err := cfg.AddProviderConfig(config); err != nil {
			fmt.Printf("❌ Failed to save provider configuration: %v\n", err)
			return
		}

		fmt.Printf("✅ Successfully added %s provider\n", strings.ToUpper(provider))
	},
}

var removeProviderCmd = &cobra.Command{
	Use:   "remove [s3|gcs]",
	Short: "Remove a cloud storage provider",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider := strings.ToLower(args[0])
		if provider != "s3" && provider != "gcs" {
			fmt.Println("❌ Invalid provider. Use 's3' or 'gcs'")
			return
		}

		// Confirm removal
		prompt := promptui.Prompt{
			Label:     fmt.Sprintf("Are you sure you want to remove %s provider", strings.ToUpper(provider)),
			IsConfirm: true,
		}
		if _, err := prompt.Run(); err != nil {
			fmt.Println("❌ Provider removal cancelled")
			return
		}

		if err := cfg.RemoveProviderConfig(provider); err != nil {
			fmt.Printf("❌ Failed to remove provider: %v\n", err)
			return
		}

		fmt.Printf("✅ Successfully removed %s provider\n", strings.ToUpper(provider))
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured cloud storage providers",
	Run: func(cmd *cobra.Command, args []string) {
		providers, err := cfg.LoadUserProviders()
		if err != nil {
			fmt.Printf("❌ Failed to load providers: %v\n", err)
			return
		}

		if len(providers.Providers) == 0 {
			fmt.Println("No providers configured")
			return
		}

		fmt.Println("Configured providers:")
		for provider, config := range providers.Providers {
			status := "✅ Enabled"
			if !config.Enabled {
				status = "❌ Disabled"
			}
			fmt.Printf("  • %s: %s\n", strings.ToUpper(provider), status)
			if config.Provider == "s3" {
				fmt.Printf("    Bucket: %s\n", config.Bucket)
				fmt.Printf("    Region: %s\n", config.Region)
			} else if config.Provider == "gcs" {
				fmt.Printf("    Project: %s\n", config.ProjectID)
				fmt.Printf("    Service Account: %s\n", filepath.Base(config.ServiceAccount))
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(providerCmd)
	providerCmd.AddCommand(addProviderCmd)
	providerCmd.AddCommand(removeProviderCmd)
	providerCmd.AddCommand(listCmd)
}

// Helper function for prompting user input
func promptLine(label string) (string, error) {
	prompt := promptui.Prompt{
		Label: label,
	}
	return prompt.Run()
}

// Helper function for prompting password
func promptPassword(label string) (string, error) {
	prompt := promptui.Prompt{
		Label: label,
		Mask:  '*',
	}
	return prompt.Run()
}
