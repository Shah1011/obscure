package cmd

import (
	"fmt"

	"github.com/shah1011/obscure/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and clear the current session",
	Run: func(cmd *cobra.Command, args []string) {
		err := config.ClearSessionEmail()
		if err != nil {
			fmt.Println("Error clearing session:", err)
			return
		}
		fmt.Println("Logged out successfully.")
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
