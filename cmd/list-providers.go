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
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configured providers
		providers, err := config.LoadUserProviders()
		if err != nil {
			return fmt.Errorf("failed to load providers: %w", err)
		}

		if len(providers.Providers) == 0 {
			fmt.Println("❌ No providers configured. Use 'obscure provider add' to add a provider.")
			return nil
		}

		// Get current session provider
		sessionProvider, err := config.GetSessionProvider()
		if err != nil {
			return fmt.Errorf("failed to get session provider: %w", err)
		}

		// Get default provider
		defaultProvider, err := config.GetUserDefaultProvider()
		if err != nil {
			return fmt.Errorf("failed to get default provider: %w", err)
		}

		// Categorize providers
		centralizedProviders := []string{}
		decentralizedProviders := []string{}

		for providerKey, providerConfig := range providers.Providers {
			if !providerConfig.Enabled {
				continue
			}

			isComplete, _ := config.IsProviderConfigComplete(providerConfig)
			if !isComplete {
				continue
			}

			switch providerConfig.Provider {
			case "s3", "gcs", "b2", "idrive", "s3-compatible":
				centralizedProviders = append(centralizedProviders, providerKey)
			case "storj":
				decentralizedProviders = append(decentralizedProviders, providerKey)
			}
		}

		fmt.Println("📋 Configured Cloud Providers:")
		fmt.Println()

		// Display Centralized Providers
		if len(centralizedProviders) > 0 {
			fmt.Println("🏢 Centralized Providers:")
			fmt.Println("─" + strings.Repeat("─", 50))
			for _, providerKey := range centralizedProviders {
				status := ""
				if providerKey == sessionProvider {
					status += " (active)"
				}
				if providerKey == defaultProvider {
					status += " (default)"
				}
				fmt.Printf("  • %s%s\n", providerKey, status)
			}
			fmt.Println()
		}

		// Display Decentralized Providers
		if len(decentralizedProviders) > 0 {
			fmt.Println("🌐 Decentralized Providers:")
			fmt.Println("─" + strings.Repeat("─", 50))
			for _, providerKey := range decentralizedProviders {
				status := ""
				if providerKey == sessionProvider {
					status += " (active)"
				}
				if providerKey == defaultProvider {
					status += " (default)"
				}
				fmt.Printf("  • %s%s\n", providerKey, status)
			}
			fmt.Println()
		}

		// Summary
		totalProviders := len(centralizedProviders) + len(decentralizedProviders)
		fmt.Printf("📊 Total: %d provider(s) configured\n", totalProviders)
		if sessionProvider != "" {
			fmt.Printf("🎯 Active session: %s\n", sessionProvider)
		}
		if defaultProvider != "" {
			fmt.Printf("⭐ Default: %s\n", defaultProvider)
		}

		return nil
	},
}

func init() {
	listProvidersCmd.Flags().BoolVar(&defaultOnly, "default", false, "Show only the default provider")
	rootCmd.AddCommand(listProvidersCmd)
}
