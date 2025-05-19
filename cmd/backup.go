package cmd

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
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

		// Prompt for password
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

		// 🧂 Generate salt
		fmt.Println("🔹 Generating salt...")
		salt, err := utils.GenerateSalt()
		if err != nil {
			fmt.Println("❌ Failed to generate salt:", err)
			return
		}
		fmt.Println("✅ Salt generated.")

		// 🔑 Derive key
		fmt.Println("🔹 Deriving encryption key...")
		key, err := utils.DeriveKey(password, salt)
		if err != nil {
			fmt.Println("❌ Key derivation failed:", err)
			return
		}
		fmt.Println("✅ Key derived.")

		// 🗜️ Zip directory
		outputZip := fmt.Sprintf("%s_v%s.zip", tag, version)
		fmt.Println("🔹 Zipping directory:", inputDir)
		err = utils.ZipDirectory(inputDir, outputZip)
		if err != nil {
			fmt.Println("❌ Failed to zip directory:", err)
			return
		}
		fmt.Println("✅ Directory zipped:", outputZip)

		// 🔐 Encrypt file
		encryptedFile := fmt.Sprintf("%s_v%s.obscure", tag, version)
		fmt.Println("🔹 Encrypting zipped file...")
		err = utils.EncryptFile(outputZip, encryptedFile, key)
		if err != nil {
			fmt.Println("❌ Failed to encrypt file:", err)
			return
		}
		fmt.Println("✅ Backup created and encrypted:", encryptedFile)

		// 🧠 Fetch user ID
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			fmt.Println("❌ Failed to load AWS config:", err)
			return
		}
		stsClient := sts.NewFromConfig(cfg)
		identity, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
		if err != nil {
			fmt.Println("❌ Failed to get AWS user identity:", err)
			return
		}
		userID := *identity.UserId

		// ☁️ Upload to S3
		s3Key := fmt.Sprintf("backups/%s/%s.obscure", userID, tag)
		fmt.Println("🔹 Uploading backup to S3 at:", s3Key)
		if err := utils.UploadToS3(s3Key, encryptedFile); err != nil {
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
	backupCmd.MarkFlagRequired("tag")
	backupCmd.MarkFlagRequired("version")
}
