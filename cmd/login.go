package cmd

import (
	"fmt"
	"os"

	cfg "github.com/shah1011/obscure/internal/config"
	"github.com/shah1011/obscure/internal/firebase"
	"github.com/shah1011/obscure/utils"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to your Obscure account",
	Run: func(cmd *cobra.Command, args []string) {
		// Step 1: Email input
		email, err := utils.PromptEmail("📧 Enter email: ")
		if err != nil || email == "" {
			fmt.Println("❌ Invalid email.")
			return
		}

		// Step 2: Password input
		password, err := utils.PromptPassword("🔏 Enter password: ")
		if err != nil || password == "" {
			fmt.Println("❌ Invalid password.")
			return
		}

		// Step 3: Firebase authentication
		apiKey := firebase.GetFirebaseApiKey()
		if apiKey == "" {
			fmt.Println("❌ Firebase API key not available. Please check configuration.")
			return
		}
		idToken, err := firebase.FirebaseLogin(email, password, apiKey)
		if err != nil {
			fmt.Println("❌ Login failed:", err)
			return
		}

		// Step 4: Get user data from Firestore
		client, err := firebase.GetFirestoreClient()
		if err != nil {
			fmt.Println("❌ Failed to connect to Firestore:", err)
			return
		}
		defer client.Close()

		iter := client.Collection("users").Where("email", "==", email).Documents(cmd.Context())
		doc, err := iter.Next()
		if err != nil {
			fmt.Println("❌ Failed to fetch user data from Firestore:", err)
			return
		}
		var userData struct {
			Username string `firestore:"username"`
			Email    string `firestore:"email"`
			Provider string `firestore:"defaultProvider"`
		}
		err = doc.DataTo(&userData)
		if err != nil {
			fmt.Println("❌ Failed to parse user data:", err)
			return
		}

		// Step 5: Save session details locally
		if err := cfg.SetSessionEmail(email); err != nil {
			fmt.Println("⚠️  Login successful but failed to save session:", err)
			return
		}
		if err := cfg.SetSessionUsername(userData.Username); err != nil {
			fmt.Println("⚠️  Login successful but failed to save session username:", err)
			return
		}
		if err := cfg.SetSessionToken(idToken); err != nil {
			fmt.Println("⚠️  Login successful but failed to save session token:", err)
			return
		}

		// Step 6: Set provider from user's configuration
		if err := cfg.SetSessionProvider(userData.Provider); err != nil {
			fmt.Println("⚠️  Login successful but failed to set active provider:", err)
		}

		fmt.Println("✅ Login successful. Welcome back,", userData.Username)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
