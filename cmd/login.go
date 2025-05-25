package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/shah1011/obscure/internal/auth"
	"github.com/shah1011/obscure/internal/config"
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

		// Check if the user has signed up
		exists, err := config.IsUserSignedUp(email)
		if err != nil {
			fmt.Println("Error checking user:", err)
			return
		}
		if !exists {
			fmt.Println("No user found with that email. Please sign up first using `obscure signup`.")
			return
		}

		// Prompt for password
		password, err := utils.PromptPassword("🔐 Enter your password: ")
		if err != nil {
			fmt.Println("❌ Failed to read password:", err)
			return
		}

		// Verify password
		valid := auth.CheckPassword(email, password)
		if !valid {
			fmt.Println("❌ Invalid password. Please try again.")
			return
		}

		username, err := config.GetUsernameByEmail(email)
		if err != nil {
			fmt.Println("Failed to get username:", err)
			return
		}

		// Save email and username in session
		err = config.SetSessionEmail(email)
		if err != nil {
			fmt.Println("Failed to save session email:", err)
			return
		}

		err = config.SetSessionUsername(username)
		if err != nil {
			fmt.Println("Failed to save session username:", err)
			return
		}

		fmt.Println("Logged in successfully as:", username)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
