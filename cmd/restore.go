package cmd

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/shah1011/obscure/utils"
	"github.com/spf13/cobra"
)

var restoreTag string
var restoreVersion string

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore an encrypted backup from S3",
	Run: func(cmd *cobra.Command, args []string) {
		// 📦 Construct expected filenames
		encryptedFile := fmt.Sprintf("%s_v%s.obscure", restoreTag, restoreVersion)
		outputZip := fmt.Sprintf("%s_v%s.zip", restoreTag, restoreVersion)
		outputDir := fmt.Sprintf("restored_%s_v%s", restoreTag, restoreVersion)

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

		// 🧬 Construct S3 key using tag and userID
		s3Key := fmt.Sprintf("backups/%s/%s.obscure", userID, restoreTag)

		// ☁️ Download encrypted backup
		fmt.Println("🔽 Downloading encrypted backup from S3...")
		err = utils.DownloadFromS3(bucketName, s3Key, encryptedFile) // using globally declared bucketName
		if err != nil {
			fmt.Println("❌ Failed to download backup:", err)
			return
		}
		fmt.Println("✅ Backup downloaded:", encryptedFile)

		// 🔐 Prompt for decryption password
		fmt.Println("🔐 Prompting for password...")
		password, err := utils.PromptPassword("🔐 Enter password for decryption: ")
		if err != nil {
			fmt.Println("❌ Failed to read password:", err)
			return
		}
		fmt.Println("✅ Password securely received.")

		// 🧂 Extract salt from encrypted file
		salt, err := utils.ExtractSaltFromEncryptedFile(encryptedFile)
		if err != nil {
			fmt.Println("❌ Failed to extract salt:", err)
			return
		}

		// 🔑 Derive key
		key, err := utils.DeriveKey(password, salt)
		if err != nil {
			fmt.Println("❌ Key derivation failed:", err)
			return
		}

		// 🔓 Decrypt file
		fmt.Println("🔓 Decrypting backup...")
		err = utils.DecryptFile(encryptedFile, outputZip, key)
		if err != nil {
			fmt.Println("❌ Failed to decrypt file:", err)
			return
		}
		fmt.Println("✅ Decrypted to:", outputZip)

		// 🗃️ Unzip
		fmt.Println("📂 Unzipping backup...")
		err = utils.UnzipFile(outputZip, outputDir)
		if err != nil {
			fmt.Println("❌ Failed to unzip:", err)
			return
		}
		fmt.Println("✅ Backup restored to:", outputDir)
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().StringVarP(&restoreTag, "tag", "t", "", "Tag of the backup to restore")
	restoreCmd.Flags().StringVarP(&restoreVersion, "version", "v", "", "Version of the backup to restore")
	restoreCmd.MarkFlagRequired("tag")
	restoreCmd.MarkFlagRequired("version")
}
