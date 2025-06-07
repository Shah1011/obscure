package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/shah1011/obscure/internal/auth"
	cfg "github.com/shah1011/obscure/internal/config"
	"github.com/shah1011/obscure/internal/firebase"
	"github.com/shah1011/obscure/utils"
	"github.com/spf13/cobra"
)

var signupCmd = &cobra.Command{
	Use:   "signup",
	Short: "Sign up for a new Obscure account",
	Run: func(cmd *cobra.Command, args []string) {
		// Step 1: Email input
		email, err := utils.PromptEmail("📧 Enter email: ")
		if err != nil || strings.TrimSpace(email) == "" {
			fmt.Println("❌ Invalid email.")
			return
		}

		// Check if user already exists
		exists, err := firebase.UserEmailExists(email)
		if err != nil {
			fmt.Println("❌ Error checking user in Firestore:", err)
			return
		}
		if exists {
			fmt.Println("❌ User already exists with that email.")
			return
		}

		// Step 2: Username input
		username, err := utils.PromptUsername("👤 Choose a username: ")
		if err != nil || strings.TrimSpace(username) == "" {
			fmt.Println("❌ Invalid username.")
			return
		}

		taken, err := firebase.UsernameTaken(username)
		if err != nil {
			fmt.Println("❌ Error checking username in Firestore:", err)
			return
		}
		if taken {
			fmt.Println("❌ Username is already taken.")
			return
		}

		// Step 3: Password input + confirm
		password, err := utils.PromptPasswordConfirm("🔏 Create password: ")
		if err != nil || strings.TrimSpace(password) == "" {
			fmt.Println("❌ Password confirmation failed.")
			return
		}

		// Step 4: Prompt for default cloud provider
		providers := []string{"Amazon S3", "Google Cloud Storage"}
		providerKeys := []string{"s3", "gcs"}

		underline := "\033[4m"
		reset := "\033[0m"
		prompt := promptui.Select{
			Label: "Select your default cloud provider",
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
			fmt.Println("❌ Provider selection cancelled")
			return
		}
		provider := providerKeys[idx]

		// Create provider config
		config := &cfg.CloudProviderConfig{
			Provider: provider,
			Enabled:  true,
		}

		// Configure the selected provider
		switch provider {
		case "s3":
			fmt.Println("\n🔧 Configure your AWS S3 storage:")
			bucket, err := utils.PromptLine("Enter S3 bucket name: ")
			if err != nil {
				fmt.Println("❌ Invalid bucket name")
				return
			}
			region, err := utils.PromptLine("Enter AWS region (e.g., us-east-1): ")
			if err != nil {
				fmt.Println("❌ Invalid region")
				return
			}
			accessKey, err := utils.PromptLine("Enter AWS Access Key ID: ")
			if err != nil {
				fmt.Println("❌ Invalid access key")
				return
			}
			secretKey, err := utils.PromptPassword("Enter AWS Secret Access Key: ")
			if err != nil {
				fmt.Println("❌ Invalid secret key")
				return
			}

			config.Bucket = bucket
			config.Region = region
			config.AccessKeyID = accessKey
			config.SecretAccessKey = secretKey

		case "gcs":
			fmt.Println("\n🔧 Configure your Google Cloud Storage:")
			projectID, err := utils.PromptLine("Enter Google Cloud Project ID: ")
			if err != nil {
				fmt.Println("❌ Invalid project ID")
				return
			}
			serviceAccountPath, err := utils.PromptLine("Enter path to service account key file: ")
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

		// Save provider configuration locally
		if err := cfg.AddProviderConfig(config); err != nil {
			fmt.Printf("❌ Failed to save provider configuration: %v\n", err)
			return
		}

		// Step 5: Send verification code
		code, err := auth.SendVerificationCode(email)
		if err != nil {
			fmt.Println("❌ Failed to send verification code:", err)
			return
		}
		fmt.Printf("📬 Verification code for %s: %s (simulated)\n", email, code)

		// Step 6: Prompt for verification code
		enteredCode, err := utils.PromptLine("🔑 Enter the verification code: ")
		if err != nil || strings.TrimSpace(enteredCode) == "" {
			fmt.Println("❌ Invalid input.")
			return
		}

		// Step 7: Verify code
		if !auth.VerifyCode(email, enteredCode) {
			fmt.Println("❌ Verification failed: invalid or expired code.")
			return
		}

		// Step 8: Create user in Firebase Auth
		userRecord, err := firebase.SignUpUser(email, password)
		if err != nil {
			fmt.Println("❌ Failed to create Firebase user:", err)
			return
		}

		// Step 9: Save user data in Firestore with provider info
		providerConfig := map[string]interface{}{
			"enabled": config.Enabled,
			"type":    config.Provider,
			"bucket":  config.Bucket,
		}
		if config.Provider == "s3" {
			providerConfig["region"] = config.Region
		} else if config.Provider == "gcs" {
			providerConfig["projectId"] = config.ProjectID
		}

		err = firebase.SaveUserData(userRecord.UID, username, provider, providerConfig)
		if err != nil {
			fmt.Println("❌ Failed to save user data in Firestore:", err)
			return
		}

		// Step 10: Login to get session token
		idToken, err := firebase.FirebaseLogin(email, password, os.Getenv("FIREBASE_API_KEY"))
		if err != nil {
			fmt.Println("❌ Login failed after signup:", err)
			return
		}

		// Save session details locally
		if err := cfg.SetSessionEmail(email); err != nil {
			fmt.Println("⚠️  Signup successful but failed to save session:", err)
			return
		}
		if err := cfg.SetSessionUsername(username); err != nil {
			fmt.Println("⚠️  Signup successful but failed to save session username:", err)
			return
		}
		if err := cfg.SetSessionToken(idToken); err != nil {
			fmt.Println("⚠️  Signup successful but failed to save session token:", err)
			return
		}
		if err := cfg.SetUserDefaultProvider(provider); err != nil {
			fmt.Println("⚠️  Signup successful but failed to save default provider:", err)
		}
		if err := cfg.SetSessionProvider(provider); err != nil {
			fmt.Println("⚠️  Signup successful but failed to set active provider:", err)
		}

		fmt.Println("✅ Signup complete. You are now registered.")
	},
}

func init() {
	rootCmd.AddCommand(signupCmd)
}
