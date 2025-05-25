package cmd

import (
	"fmt"

	"github.com/shah1011/obscure/internal/config"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the currently logged in user",
	Run: func(cmd *cobra.Command, args []string) {
		email, err := config.GetSessionEmail()
		if err != nil || email == "" {
			fmt.Println("👤 No user is currently logged in.")
			return
		}

		username, err := config.GetSessionUsername()
		if err != nil || username == "" {
			fmt.Println("👤 Logged in as:", email)
			return
		}

		fmt.Printf("👤 Logged in as: %s (%s)\n", username, email)
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
