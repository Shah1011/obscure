package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/shah1011/obscure/internal/auth"
	"github.com/shah1011/obscure/internal/config"
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
		if auth.UserExists(email) {
			fmt.Println("❌ User already exists with that email.")
			return
		}

		// Step 2: Username input
		username, err := utils.PromptUsername("👤 Choose a username: ")
		if err != nil || strings.TrimSpace(username) == "" {
			fmt.Println("❌ Invalid username.")
			return
		}

		if auth.UsernameExists(username) {
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
			Label: "☁️  Choose your default cloud provider:",
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
			fmt.Println("❌ Cloud selection failed:", err)
			return
		}
		provider := providerKeys[idx]
		if err != nil || strings.TrimSpace(provider) == "" {
			fmt.Println("❌ Invalid cloud provider selection.")
			return
		}

		// Step 5: Send simulated verification code
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

		// Step 8: Save user
		err = auth.SaveUser(email, username, password)
		if err != nil {
			fmt.Println("❌ Failed to save user:", err)
			return
		}

		fmt.Println("✅ Signup complete. You are now registered.")

		// Step 9: Save session details
		if err := config.SetSessionEmail(email); err != nil {
			fmt.Println("⚠️  Signup successful but failed to save session:", err)
			return
		}
		if err := config.SetSessionUsername(username); err != nil {
			fmt.Println("⚠️  Signup successful but failed to save session username:", err)
			return
		}

		// ✅ Step 10: Save default provider for the user
		if err := config.SetUserDefaultProvider(provider); err != nil {
			fmt.Println("⚠️  Signup successful but failed to save default provider:", err)
		}

		// ✅ Step 11: Set active session provider
		if err := config.SetSessionProvider(provider); err != nil {
			fmt.Println("⚠️  Signup successful but failed to set active provider:", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(signupCmd)
}
