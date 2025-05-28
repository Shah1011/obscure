package cmd

import (
	"fmt"
	"strings"

	"github.com/shah1011/obscure/internal/config"
	"github.com/spf13/cobra"
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debug authentication and token storage",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔍 Obscure CLI Debug Information")
		fmt.Println("================================")

		// Debug users.json structure
		config.DebugUsersJSON()

		fmt.Println("\n🔍 Testing token retrieval...")

		// Test current GetSessionToken
		token, err := config.GetSessionToken()
		if err != nil {
			fmt.Printf("❌ GetSessionToken failed: %v\n", err)
		} else {
			fmt.Printf("✅ GetSessionToken succeeded (length: %d)\n", len(token))

			// Quick token validation
			if strings.Count(token, ".") == 2 {
				fmt.Println("✅ Token is in JWT format")
			} else {
				fmt.Println("⚠️  Token is not in JWT format")
			}
		}

		// Test other session data
		fmt.Println("\n🔍 Testing other session data...")

		email, err := config.GetSessionEmail()
		if err != nil {
			fmt.Printf("❌ GetSessionEmail failed: %v\n", err)
		} else {
			fmt.Printf("✅ Email: %s\n", email)
		}

		username, err := config.GetSessionUsername()
		if err != nil {
			fmt.Printf("❌ GetSessionUsername failed: %v\n", err)
		} else {
			fmt.Printf("✅ Username: %s\n", username)
		}

		provider, err := config.GetSessionProvider()
		if err != nil {
			fmt.Printf("❌ GetSessionProvider failed: %v\n", err)
		} else {
			fmt.Printf("✅ Provider: %s\n", provider)
		}

		fmt.Println("\n💡 If token is expired, run: obscure logout && obscure login")
	},
}

func init() {
	rootCmd.AddCommand(debugCmd)
}
