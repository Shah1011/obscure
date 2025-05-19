package cmd

import (
	"fmt"

	"github.com/shah1011/obscure/utils"
	"github.com/spf13/cobra"
)

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use:   "backup [directory]",
	Args:  cobra.ExactArgs(1),
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		tag, _ := cmd.Flags().GetString("tag")
		version, _ := cmd.Flags().GetString("version")
		inputDir := args[0]
		outputZip := fmt.Sprintf("%s_v%s.zip", tag, version)
		encryptedFile := fmt.Sprintf("%s_v%s.obscure", tag, version)

		fmt.Println("🔐 Prompting for password...")
		password, err := utils.PromptPassword("🔐 Enter password for encryption: ")
		if err != nil {
			fmt.Println("❌ Failed to read password:", err)
			return
		}
		fmt.Println("✅ Password securely received.")

		fmt.Println("🔹 Generating salt...")
		salt, err := utils.GenerateSalt()
		if err != nil {
			fmt.Println("❌ Failed to generate salt:", err)
			return
		}
		fmt.Println("✅ Salt generated.")

		fmt.Println("🔹 Deriving encryption key...")
		key, err := utils.DeriveKey(password, salt)
		if err != nil {
			fmt.Println("❌ Key derivation failed:", err)
			return
		}
		fmt.Println("✅ Key derived.")

		fmt.Println("🔹 Zipping directory:", inputDir)
		err = utils.ZipDirectory(inputDir, outputZip)
		if err != nil {
			fmt.Println("❌ Failed to zip directory:", err)
			return
		}
		fmt.Println("✅ Directory zipped:", outputZip)

		fmt.Println("🔹 Encrypting zipped file...")
		err = utils.EncryptFile(outputZip, encryptedFile, key)
		if err != nil {
			fmt.Println("❌ Failed to encrypt file:", err)
			return
		}
		fmt.Println("✅ Backup created and encrypted:", encryptedFile)

		fmt.Println("🔹 Uploading backup to S3...")
		err = utils.UploadToS3("Backup", encryptedFile)
		if err != nil {
			fmt.Println("❌ Upload failed:", err)
			return
		}
		fmt.Println("✅ Backup uploaded to S3")

		fmt.Println("✅ Backup completed successfully")
	},
}

var tag string
var version string

func init() {
	rootCmd.AddCommand(backupCmd)

	backupCmd.Flags().StringVarP(&tag, "tag", "t", "", "Tag for the backup")
	backupCmd.Flags().StringVarP(&version, "version", "v", "", "Version for the backup")
	backupCmd.MarkFlagRequired("tag")
	backupCmd.MarkFlagRequired("version")
}
