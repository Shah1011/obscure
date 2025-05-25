package cmd

import (
	"fmt"
	"strings"

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
		// Check if user or username exists
		if auth.UserExists(email) {
			fmt.Println("❌ User already exists with that email.")
			return
		}
		if auth.UsernameExists(username) {
			fmt.Println("❌ Username is already taken.")
			return
		}

		// Step 2: Password input + confirm
		password, err := utils.PromptPasswordConfirm("🔏 Create password: ")
		if err != nil || strings.TrimSpace(password) == "" {
			fmt.Println("❌ Password confirmation failed.")
			return
		}

		// Step 3: Send simulated verification code
		code, err := auth.SendVerificationCode(email)
		if err != nil {
			fmt.Println("❌ Failed to send verification code:", err)
			return
		}

		fmt.Printf("📬 Verification code for %s: %s (simulated)\n", email, code)

		// Step 4: Prompt for code input
		enteredCode, err := utils.PromptLine("🔑 Enter the verification code: ")
		if err != nil || strings.TrimSpace(enteredCode) == "" {
			fmt.Println("❌ Invalid input.")
			return
		}

		// Step 5: Verify code
		if !auth.VerifyCode(email, enteredCode) {
			fmt.Println("❌ Verification failed: invalid or expired code.")
			return
		}

		// Step 6: Save user
		err = auth.SaveUser(email, username, password)
		if err != nil {
			fmt.Println("❌ Failed to save user:", err)
			return
		}

		fmt.Println("✅ Signup complete. You are now registered.")

		err = config.SetSessionEmail(email)
		if err != nil {
			fmt.Println("⚠️  Signup successful but failed to save session:", err)
			return
		}
		err = config.SetSessionUsername(username)
		if err != nil {
			fmt.Println("⚠️  Signup successful but failed to save session username:", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(signupCmd)
}
