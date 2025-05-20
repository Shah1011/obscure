package cmd

import (
	"fmt"
	"os"

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
		userID, err := utils.GetUserID()
		if err != nil {
			fmt.Println("❌ Failed to get AWS user ID:", err)
			return
		}

		// 🧬 Construct S3 key using tag and userID
		s3Key := fmt.Sprintf("backups/%s/%s_v%s.obscure", userID, restoreTag, restoreVersion)

		// ☁️ Download encrypted backup
		fmt.Println("🔽 Downloading encrypted backup from S3...")

		file, err := os.Create(encryptedFile)
		if err != nil {
			fmt.Println("❌ Failed to create file:", err)
			return
		}
		defer file.Close()

		progressWriter := utils.NewProgressWriter(file, "Downloading...", 40, -1)

		err = utils.DownloadFromS3(bucketName, s3Key, progressWriter)
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

		// // 🧂 Extract salt from encrypted file
		// salt, err := utils.ExtractSaltFromEncryptedFile(encryptedFile)
		// if err != nil {
		// 	fmt.Println("❌ Failed to extract salt:", err)
		// 	return
		// }

		// // 🔑 Derive key
		// key, err := utils.DeriveKey(password, salt)
		// if err != nil {
		// 	fmt.Println("❌ Key derivation failed:", err)
		// 	return
		// }

		// 🔓 Decrypt file
		fmt.Println("🔓 Decrypting backup...")
		err = utils.DecryptFile(encryptedFile, outputZip, password)
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
