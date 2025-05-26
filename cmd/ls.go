package cmd

// import (
// 	"fmt"

// 	"github.com/shah1011/obscure/internal/config"
// 	"github.com/spf13/cobra"
// )

// var lsCmd = &cobra.Command{
// 	Use:   "ls",
// 	Short: "List all backups",
// 	Run: func(cmd *cobra.Command, args []string) {
// 		userFlag, _ := cmd.Flags().GetString("user")
// 		var userID string
// 		var err error

// 		if userFlag != "" {
// 			userID, err = config.GetUserID(userFlag)
// 			if err != nil {
// 				fmt.Println("Error getting user ID:", err)
// 				return
// 			}
// 		} else {
// 			userID, err = config.GetUserID(config.GetSessionEmail())

// 		}
// 	},
// }
