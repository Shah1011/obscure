package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/shah1011/obscure/internal/config"
	firebase "github.com/shah1011/obscure/internal/firebase"
	"github.com/shah1011/obscure/utils"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in with your email and start a session",
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter your email: ")
		email, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return
		}
		email = strings.TrimSpace(email)
		if email == "" {
			fmt.Println("Email cannot be empty.")
			return
		}

		// Prompt for password
		password, err := utils.PromptPassword("🔐 Enter your password: ")
		if err != nil {
			fmt.Println("❌ Failed to read password:", err)
			return
		}

		// Call Firebase REST API to login and get idToken
		apiKey := os.Getenv("FIREBASE_API_KEY")
		if apiKey == "" {
			fmt.Println("❌ FIREBASE_API_KEY environment variable is not set")
			return
		}
		idToken, err := firebase.FirebaseLogin(email, password, apiKey)
		if err != nil {
			fmt.Println("❌ Login failed:", err)
			return
		}

		// Debug: Print token length to verify we got something
		fmt.Printf("🔑 Got token of length: %d\n", len(idToken))

		// Save token locally
		if err := config.SetSessionToken(idToken); err != nil {
			fmt.Println("⚠️  Login successful but failed to save session token:", err)
			return
		}

		// Verify token was saved
		savedToken, err := config.GetSessionToken()
		if err != nil || savedToken == "" {
			fmt.Println("⚠️  Token was not saved correctly")
			return
		}
		if savedToken != idToken {
			fmt.Println("⚠️  Saved token does not match received token")
			return
		}

		// Use idToken to fetch user info from Firestore (optional)
		userData, err := config.GetUserDataByEmail(email)
		if err != nil {
			fmt.Println("❌ Failed to fetch user data:", err)
			return
		}

		// Save session details locally
		err = config.SetSessionEmail(email)
		if err != nil {
			fmt.Println("Failed to save session email:", err)
			return
		}
		err = config.SetSessionUsername(userData.Username)
		if err != nil {
			fmt.Println("Failed to save session username:", err)
			return
		}

		fmt.Printf("✅ Logged in successfully as %s\n", userData.Username)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
