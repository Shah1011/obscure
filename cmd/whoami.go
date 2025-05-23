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
		fmt.Println("👤 Logged in as:", email)
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
