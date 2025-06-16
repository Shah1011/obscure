package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	cfg "github.com/shah1011/obscure/internal/config"
	"github.com/spf13/cobra"
)

// Helper function to read input that handles pasting properly
func readInput(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage cloud storage providers",
	Long: `Manage your cloud storage providers. You can add, remove, and configure
your own cloud storage accounts for backup storage.`,
}

// Validation functions
func validateBucketName(bucket string) error {
	if strings.TrimSpace(bucket) == "" {
		return fmt.Errorf("bucket name cannot be empty")
	}
	if len(bucket) < 3 {
		return fmt.Errorf("bucket name must be at least 3 characters long")
	}
	if len(bucket) > 63 {
		return fmt.Errorf("bucket name must be less than 63 characters")
	}
	// Check for valid characters (alphanumeric, hyphens, dots)
	for _, char := range bucket {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '.') {
			return fmt.Errorf("bucket name contains invalid characters (only lowercase letters, numbers, hyphens, and dots allowed)")
		}
	}
	return nil
}

func validateRegion(region string) error {
	if strings.TrimSpace(region) == "" {
		return fmt.Errorf("region cannot be empty")
	}
	if len(region) < 3 {
		return fmt.Errorf("region must be at least 3 characters long")
	}
	return nil
}

func validateAccessKey(accessKey string) error {
	if strings.TrimSpace(accessKey) == "" {
		return fmt.Errorf("access key cannot be empty")
	}
	if len(accessKey) < 16 {
		return fmt.Errorf("access key must be at least 16 characters long")
	}
	return nil
}

func validateSecretKey(secretKey string) error {
	if strings.TrimSpace(secretKey) == "" {
		return fmt.Errorf("secret key cannot be empty")
	}
	if len(secretKey) < 16 {
		return fmt.Errorf("secret key must be at least 16 characters long")
	}
	return nil
}

func validateProjectID(projectID string) error {
	if strings.TrimSpace(projectID) == "" {
		return fmt.Errorf("project ID cannot be empty")
	}
	if len(projectID) < 3 {
		return fmt.Errorf("project ID must be at least 3 characters long")
	}
	return nil
}

func validateServiceAccountPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("service account path cannot be empty")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("service account file not found at: %s", path)
	}
	return nil
}

func validateEndpoint(endpoint string) error {
	if strings.TrimSpace(endpoint) == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}
	if !strings.HasPrefix(endpoint, "https://") {
		return fmt.Errorf("endpoint must start with https://")
	}
	_, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}
	return nil
}

func validateApplicationKeyID(appKeyID string) error {
	if strings.TrimSpace(appKeyID) == "" {
		return fmt.Errorf("application key ID cannot be empty")
	}
	if len(appKeyID) < 8 {
		return fmt.Errorf("application key ID must be at least 8 characters long")
	}
	return nil
}

func validateApplicationKey(appKey string) error {
	if strings.TrimSpace(appKey) == "" {
		return fmt.Errorf("application key cannot be empty")
	}
	if len(appKey) < 8 {
		return fmt.Errorf("application key must be at least 8 characters long")
	}
	return nil
}

// Check if provider configuration is complete
func isProviderConfigComplete(config *cfg.CloudProviderConfig) (bool, []string) {
	var missing []string

	switch config.Provider {
	case "s3":
		if config.Bucket == "" {
			missing = append(missing, "bucket name")
		}
		if config.Region == "" {
			missing = append(missing, "region")
		}
		if config.AccessKeyID == "" {
			missing = append(missing, "access key ID")
		}
		if config.SecretAccessKey == "" {
			missing = append(missing, "secret access key")
		}
	case "gcs":
		if config.ProjectID == "" {
			missing = append(missing, "project ID")
		}
		if config.ServiceAccount == "" {
			missing = append(missing, "service account path")
		}
	case "b2":
		if config.Bucket == "" {
			missing = append(missing, "bucket name")
		}
		if config.Endpoint == "" {
			missing = append(missing, "endpoint")
		}
		if config.ApplicationKeyID == "" {
			missing = append(missing, "application key ID")
		}
		if config.ApplicationKey == "" {
			missing = append(missing, "application key")
		}
	case "idrive":
		if config.Bucket == "" {
			missing = append(missing, "bucket name")
		}
		if config.Region == "" {
			missing = append(missing, "region")
		}
		if config.AccessKeyID == "" {
			missing = append(missing, "access key ID")
		}
		if config.SecretAccessKey == "" {
			missing = append(missing, "secret access key")
		}
		if config.IDriveEndpoint == "" {
			missing = append(missing, "IDrive E2 endpoint")
		}
	}

	return len(missing) == 0, missing
}

var addProviderCmd = &cobra.Command{
	Use:   "add [s3|gcs|b2|idrive]",
	Short: "Add a new cloud storage provider",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider := strings.ToLower(args[0])
		if provider != "s3" && provider != "gcs" && provider != "b2" && provider != "idrive" {
			fmt.Println("❌ Invalid provider. Use 's3', 'gcs', 'b2', or 'idrive'")
			return
		}

		config := &cfg.CloudProviderConfig{
			Provider: provider,
			Enabled:  false, // Start as disabled until fully configured
		}

		// Check if provider already exists
		existingProviders, err := cfg.LoadUserProviders()
		if err == nil {
			if existingConfig, exists := existingProviders.Providers[provider]; exists {
				// Check if existing config is complete
				isComplete, missing := isProviderConfigComplete(existingConfig)
				if isComplete {
					fmt.Printf("⚠️  Provider %s is already configured.\n", strings.ToUpper(provider))
					fmt.Print("Do you want to update the configuration? (y/N): ")
					var input string
					fmt.Scanln(&input)
					input = strings.TrimSpace(strings.ToLower(input))
					if input != "y" && input != "yes" {
						fmt.Println("❌ Configuration update cancelled.")
						return
					}
				} else {
					fmt.Printf("⚠️  Provider %s is configured but incomplete.\n", strings.ToUpper(provider))
					fmt.Printf("   Missing: %s\n", strings.Join(missing, ", "))
					fmt.Print("Do you want to update the configuration? (y/N): ")
					var input string
					fmt.Scanln(&input)
					input = strings.TrimSpace(strings.ToLower(input))
					if input != "y" && input != "yes" {
						fmt.Println("❌ Configuration update cancelled.")
						return
					}
				}
			}
		}

		// Configure provider based on type
		var configErr error
		switch provider {
		case "s3":
			configErr = configureS3Provider(config)
		case "gcs":
			configErr = configureGCSProvider(config)
		case "b2":
			configErr = configureB2Provider(config)
		case "idrive":
			configErr = configureIDriveProvider(config)
		}

		if configErr != nil {
			fmt.Printf("❌ Configuration failed: %v\n", configErr)
			return
		}

		// Check if configuration is complete
		isComplete, missing := isProviderConfigComplete(config)
		if isComplete {
			config.Enabled = true
		}

		// Save configuration
		if err := cfg.AddProviderConfig(config); err != nil {
			fmt.Printf("❌ Failed to save configuration: %v\n", err)
			return
		}

		if config.Enabled {
			fmt.Printf("✅ Successfully added and enabled %s provider\n", strings.ToUpper(provider))
		} else {
			fmt.Printf("⚠️  %s provider added but disabled due to incomplete configuration\n", strings.ToUpper(provider))
			fmt.Printf("   Missing: %s\n", strings.Join(missing, ", "))
			fmt.Printf("   Run './obscure provider add %s' again to complete configuration\n", provider)
		}
	},
}

func configureS3Provider(config *cfg.CloudProviderConfig) error {
	fmt.Println("\n🔧 Configure AWS S3 storage:")

	// Bucket name
	bucket := readInput("Enter S3 bucket name: ")
	if err := validateBucketName(bucket); err != nil {
		return fmt.Errorf("invalid bucket name: %v", err)
	}

	// Region
	region := readInput("Enter AWS region (e.g., us-east-1): ")
	if err := validateRegion(region); err != nil {
		return fmt.Errorf("invalid region: %v", err)
	}

	// Access Key ID
	accessKey := readInput("Enter AWS Access Key ID: ")
	if err := validateAccessKey(accessKey); err != nil {
		return fmt.Errorf("invalid access key: %v", err)
	}

	// Secret Access Key
	secretKey := readInput("Enter AWS Secret Access Key: ")
	if err := validateSecretKey(secretKey); err != nil {
		return fmt.Errorf("invalid secret key: %v", err)
	}

	config.Bucket = bucket
	config.Region = region
	config.AccessKeyID = accessKey
	config.SecretAccessKey = secretKey

	return nil
}

func configureGCSProvider(config *cfg.CloudProviderConfig) error {
	fmt.Println("\n🔧 Configure Google Cloud Storage:")

	// Project ID
	projectID := readInput("Enter Google Cloud Project ID: ")
	if err := validateProjectID(projectID); err != nil {
		return fmt.Errorf("invalid project ID: %v", err)
	}

	// Service Account Path
	serviceAccountPath := readInput("Enter path to service account key file: ")
	if err := validateServiceAccountPath(serviceAccountPath); err != nil {
		return fmt.Errorf("invalid service account path: %v", err)
	}

	config.ProjectID = projectID
	config.ServiceAccount = serviceAccountPath

	return nil
}

func configureB2Provider(config *cfg.CloudProviderConfig) error {
	fmt.Println("\n🔧 Configure Backblaze B2 storage:")

	// Bucket name
	bucket := readInput("Enter B2 bucket name: ")
	if err := validateBucketName(bucket); err != nil {
		return fmt.Errorf("invalid bucket name: %v", err)
	}

	// Endpoint
	endpoint := readInput("Enter B2 endpoint URL (e.g., https://s3.us-west-002.backblazeb2.com): ")
	if err := validateEndpoint(endpoint); err != nil {
		return fmt.Errorf("invalid endpoint: %v", err)
	}

	// Application Key ID
	appKeyID := readInput("Enter B2 Application Key ID: ")
	if err := validateApplicationKeyID(appKeyID); err != nil {
		return fmt.Errorf("invalid application key ID: %v", err)
	}

	// Application Key
	appKey := readInput("Enter B2 Application Key: ")
	if err := validateApplicationKey(appKey); err != nil {
		return fmt.Errorf("invalid application key: %v", err)
	}

	config.Bucket = bucket
	config.Endpoint = endpoint
	config.ApplicationKeyID = appKeyID
	config.ApplicationKey = appKey

	return nil
}

func configureIDriveProvider(config *cfg.CloudProviderConfig) error {
	fmt.Println("\n🔧 Configure IDrive E2 storage:")

	// Bucket name
	bucket := readInput("Enter IDrive E2 bucket name: ")
	if err := validateBucketName(bucket); err != nil {
		return fmt.Errorf("invalid bucket name: %v", err)
	}

	// Region
	region := readInput("Enter IDrive E2 region (e.g., us-east-1): ")
	if err := validateRegion(region); err != nil {
		return fmt.Errorf("invalid region: %v", err)
	}

	// Access Key ID
	accessKey := readInput("Enter IDrive E2 access key ID: ")
	if err := validateAccessKey(accessKey); err != nil {
		return fmt.Errorf("invalid access key: %v", err)
	}

	// Secret Access Key
	secretKey := readInput("Enter IDrive E2 secret access key: ")
	if err := validateSecretKey(secretKey); err != nil {
		return fmt.Errorf("invalid secret key: %v", err)
	}

	// Endpoint
	endpoint := readInput("Enter IDrive E2 endpoint URL (e.g., https://api.idrive.com): ")
	if err := validateEndpoint(endpoint); err != nil {
		return fmt.Errorf("invalid endpoint: %v", err)
	}

	config.Bucket = bucket
	config.Region = region
	config.AccessKeyID = accessKey
	config.SecretAccessKey = secretKey
	config.IDriveEndpoint = endpoint

	return nil
}

var removeProviderCmd = &cobra.Command{
	Use:   "remove [s3|gcs|b2|idrive]",
	Short: "Remove a cloud storage provider",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider := strings.ToLower(args[0])
		if provider != "s3" && provider != "gcs" && provider != "b2" && provider != "idrive" {
			fmt.Println("❌ Invalid provider. Use 's3', 'gcs', 'b2', or 'idrive'")
			return
		}

		// Check if provider exists
		existingProviders, err := cfg.LoadUserProviders()
		if err != nil {
			fmt.Printf("❌ Failed to load providers: %v\n", err)
			return
		}

		if _, exists := existingProviders.Providers[provider]; !exists {
			fmt.Printf("❌ Provider %s is not configured\n", strings.ToUpper(provider))
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
			// Check if configuration is complete
			isComplete, missing := isProviderConfigComplete(config)

			var status string
			if !config.Enabled {
				status = "❌ Disabled"
			} else if !isComplete {
				status = "⚠️  Incomplete"
			} else {
				status = "✅ Enabled"
			}

			fmt.Printf("  • %s: %s\n", strings.ToUpper(provider), status)

			if !isComplete {
				fmt.Printf("    Missing: %s\n", strings.Join(missing, ", "))
			}

			if config.Provider == "s3" {
				fmt.Printf("    Bucket: %s\n", config.Bucket)
				fmt.Printf("    Region: %s\n", config.Region)
			} else if config.Provider == "gcs" {
				fmt.Printf("    Project: %s\n", config.ProjectID)
				fmt.Printf("    Service Account: %s\n", filepath.Base(config.ServiceAccount))
			} else if config.Provider == "b2" {
				fmt.Printf("    Bucket: %s\n", config.Bucket)
				fmt.Printf("    Endpoint: %s\n", config.Endpoint)
			} else if config.Provider == "idrive" {
				fmt.Printf("    Bucket: %s\n", config.Bucket)
				fmt.Printf("    Region: %s\n", config.Region)
				fmt.Printf("    Endpoint: %s\n", config.IDriveEndpoint)
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
