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
		centralizedProviders := []string{"Amazon S3", "Google Cloud Storage", "Backblaze B2", "IDrive E2", "S3-compatible"}
		centralizedKeys := []string{"s3", "gcs", "b2", "idrive", "s3-compatible"}
		decentralizedProviders := []string{"Storj", "Filebase + IPFS"}
		decentralizedKeys := []string{"storj", "filebase-ipfs"}

		// For flag mode, keep old logic
		providerKeys := append(centralizedKeys, decentralizedKeys...)
		providers := append(centralizedProviders, decentralizedProviders...)

		// Get flag value
		defaultFlag, err := cmd.Flags().GetString("default")
		if err != nil {
			return err
		}
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
				return errors.New("invalid --default provider value; allowed: s3, gcs, b2, idrive, s3-compatible, or storj")
			}
		} else {
			// Two-level menu system
			underline := "\033[4m"
			reset := "\033[0m"

			for {
				// First level: Choose provider type
				providerTypes := []string{"Centralized Providers", "Decentralized Providers"}
				typePrompt := promptui.Select{
					Label: "Select provider type",
					Items: providerTypes,
					Templates: &promptui.SelectTemplates{
						Label:    "{{ . }}",
						Active:   underline + "{{ . | green }}" + reset,
						Inactive: "{{ . }}",
						Selected: "",
					},
					Stdout: os.Stderr,
				}

				typeIdx, _, err := typePrompt.Run()
				if err != nil {
					return fmt.Errorf("provider type selection cancelled or failed: %w", err)
				}

				// Second level: Choose specific provider
				var selectedProviders []string
				var selectedKeys []string

				if typeIdx == 0 {
					// Centralized providers
					selectedProviders = centralizedProviders
					selectedKeys = centralizedKeys
				} else {
					// Decentralized providers
					selectedProviders = decentralizedProviders
					selectedKeys = decentralizedKeys
				}

				// Add "Back" option to the provider list
				selectedProviders = append(selectedProviders, "← Back to provider types")
				selectedKeys = append(selectedKeys, "back")

				providerPrompt := promptui.Select{
					Label: "Select provider",
					Items: selectedProviders,
					Templates: &promptui.SelectTemplates{
						Label:    "{{ . }}",
						Active:   underline + "{{ . | green }}" + reset,
						Inactive: "{{ . }}",
						Selected: "",
					},
					Stdout: os.Stderr,
				}

				providerIdx, _, err := providerPrompt.Run()
				if err != nil {
					return fmt.Errorf("provider selection cancelled or failed: %w", err)
				}

				// Check if user selected "Back"
				if selectedKeys[providerIdx] == "back" {
					// Continue the loop to go back to provider type selection
					continue
				}

				// User selected a valid provider
				newProviderKey = selectedKeys[providerIdx]
				newProviderName = selectedProviders[providerIdx]
				break
			}
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
