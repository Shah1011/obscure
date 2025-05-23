package cmd

import (
	"fmt"

	"github.com/shah1011/obscure/internal/auth"
	"github.com/shah1011/obscure/internal/config"
	"github.com/shah1011/obscure/utils"
	"github.com/spf13/cobra"
)

var tag string
var version string
var bucketName = "obscure-open" // change this to your actual bucket

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup [directory]",
	Args:  cobra.ExactArgs(1),
	Short: "Back up and encrypt a directory, then upload to S3",
	Run: func(cmd *cobra.Command, args []string) {
		inputDir := args[0]

		// 🧠 Get AWS user identity
		userFlag, _ := cmd.Flags().GetString("user")
		var userID string

		if userFlag != "" {
			userID = userFlag
		} else {
			var err error
			userID, err = config.GetSessionEmail()
			if err != nil || userID == "" {
				fmt.Println("❌ You are not logged in. Use --user or run `obscure login`")
				return
			}
		}

		// 🔐 Prompt for password
		password, err := utils.PromptPassword("🔐 Enter password for encryption: ")
		if err != nil {
			fmt.Println("❌ Failed to read password:", err)
			return
		}
		confirmPassword, err := utils.PromptPassword("🔐 Re-enter password to confirm: ")
		if err != nil {
			fmt.Println("❌ Failed to read password confirmation:", err)
			return
		}
		if password != confirmPassword {
			fmt.Println("❌ Passwords do not match. Please try again.")
			return
		}
		fmt.Println("✅ Password securely confirmed.")

		// 🗜️ Zip directory to buffer
		fmt.Println("🔹 Zipping directory:", inputDir)
		zipBuffer, err := utils.ZipDirectoryToBuffer(inputDir)
		if err != nil {
			fmt.Println("❌ Failed to zip directory:", err)
			return
		}
		fmt.Println("✅ Directory zipped in-memory.")

		// 🔐 Encrypt zipped buffer
		fmt.Println("🔹 Encrypting zipped data...")
		encryptedData, err := utils.EncryptBuffer(zipBuffer, password)
		if err != nil {
			fmt.Println("❌ Failed to encrypt data:", err)
			return
		}
		fmt.Println("✅ Data encrypted in-memory.")

		// ☁️ Upload to S3
		username, err := auth.GetUsernameByEmail(userID)
		if err != nil {
			fmt.Println("❌ Failed to get username for S3 path:", err)
			return
		}
		s3Key := fmt.Sprintf("backups/%s/%s_v%s.obscure", username, tag, version)
		fmt.Println("🔹 Uploading backup to S3 at:", s3Key)
		progressReader := utils.NewProgressBuffer(encryptedData.Bytes(), "Uploading...", 40)
		err = utils.UploadToS3(progressReader, bucketName, s3Key)
		if err != nil {
			fmt.Printf("❌ Upload failed: %v\n", err)
			return
		}
		fmt.Println("✅ Backup uploaded to S3")
		fmt.Println("🎉 Backup completed successfully!")
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVarP(&tag, "tag", "t", "", "Tag for the backup")
	backupCmd.Flags().StringVarP(&version, "version", "v", "", "Version for the backup")
	backupCmd.Flags().String("user", "", "Email to identify backup owner (optional if logged in)")
	backupCmd.MarkFlagRequired("tag")
	backupCmd.MarkFlagRequired("version")
}
