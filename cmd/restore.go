/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/shah1011/obscure/utils"
	"github.com/spf13/cobra"
)

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		tag, _ := cmd.Flags().GetString("tag")
		version, _ := cmd.Flags().GetString("version")
		encryptedFile := fmt.Sprintf("%s_v%s.obscure", tag, version)
		decryptedZip := fmt.Sprintf("%s_v%s.zip", tag, version)
		restoredDir := fmt.Sprintf("%s_v%s_restored", tag, version)

		fmt.Println("🔐 Prompting for password...")
		password, err := utils.PromptPassword("🔐 Enter password for decryption: ")
		if err != nil {
			fmt.Println("❌ Failed to read password:", err)
			return
		}
		fmt.Println("✅ Password securely received.")

		fmt.Println("🔽 Downloading encrypted backup from S3...")
		err = utils.DownloadFromS3("your-bucket-name", encryptedFile, encryptedFile)
		if err != nil {
			fmt.Println("❌ Failed to download backup:", err)
			return
		}
		fmt.Println("✅ Backup downloaded:", encryptedFile)

		fmt.Println("🔹 Reading salt from encrypted file...")
		salt, err := utils.ExtractSaltFromEncryptedFile(encryptedFile)
		if err != nil {
			fmt.Println("❌ Failed to extract salt:", err)
			return
		}
		fmt.Println("✅ Salt extracted.")

		fmt.Println("🔹 Deriving encryption key...")
		key, err := utils.DeriveKey(password, salt)
		if err != nil {
			fmt.Println("❌ Key derivation failed:", err)
			return
		}
		fmt.Println("✅ Key derived.")

		fmt.Println("🔓 Decrypting backup file...")
		err = utils.DecryptFile(encryptedFile, decryptedZip, key)
		if err != nil {
			fmt.Println("❌ Failed to decrypt file:", err)
			return
		}
		fmt.Println("✅ Backup decrypted:", decryptedZip)

		fmt.Println("📦 Unzipping backup...")
		err = utils.UnzipFile(decryptedZip, restoredDir)
		if err != nil {
			fmt.Println("❌ Failed to unzip backup:", err)
			return
		}
		fmt.Println("✅ Backup restored to directory:", restoredDir)
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().StringVarP(&tag, "tag", "t", "", "Tage for restore")
	restoreCmd.Flags().StringVarP(&version, "version", "v", "", "Version to restore")
	restoreCmd.MarkFlagRequired("tag")
	restoreCmd.MarkFlagRequired("version")
}
