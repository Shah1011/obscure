package cmd

import (
	"fmt"
	"strings"

	"github.com/shah1011/obscure/internal/config"
	"github.com/shah1011/obscure/utils"
	"github.com/spf13/cobra"
)

var restoreTag string
var restoreVersion string

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore an encrypted backup from S3",
	Run: func(cmd *cobra.Command, args []string) {
		// ☁️ Get AWS user ID
		userFlag, _ := cmd.Flags().GetString("user")
		var userID string
		var err error

		if userFlag != "" {
			userID = userFlag
		} else {
			userID, err = config.GetSessionEmail()
			if err != nil || userID == "" {
				fmt.Println("❌ You are not logged in. Use --user or run `obscure login`")
				return
			}
		}

		// 🧬 Construct S3 key
		s3Key := fmt.Sprintf("backups/%s/%s_v%s.obscure", userID, restoreTag, restoreVersion)
		outputDir := fmt.Sprintf("restored_%s_v%s", restoreTag, restoreVersion)

		// 🔐 Prompt for decryption password
		password, err := utils.PromptPassword("🔐 Enter decryption password:")
		if err != nil || strings.TrimSpace(password) == "" {
			fmt.Println("❌ Invalid or empty password.")
			return
		}

		// 📦 Get object size for progress bar
		size, err := utils.GetObjectSize(bucketName, s3Key)
		if err != nil {
			fmt.Println("❌ Could not get backup size:", err)
			return
		}

		// 📥 Stream download with progress
		fmt.Println("🔽 Downloading encrypted backup from S3...")
		rawReader, err := utils.DownloadFromS3Stream(bucketName, s3Key)
		if err != nil {
			fmt.Println("❌ Failed to download backup:", err)
			return
		}
		defer rawReader.Close()

		progressReader := utils.NewProgressReader(rawReader, "Downloading", 40, size)

		// 🔓 Decrypt stream
		decStream, err := utils.DecryptStream(progressReader, password)
		if err != nil {
			fmt.Println("❌ Decryption failed:", err)
			return
		}

		// 📂 Unzip stream directly to restore folder
		fmt.Println("📂 Restoring files...")
		err = utils.DecompressZstdToDirectory(decStream, outputDir)
		if err != nil {
			fmt.Println("❌ Failed to unzip:", err)
			return
		}

		fmt.Println("✅ Restore complete at:", outputDir)
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().StringVarP(&restoreTag, "tag", "t", "", "Tag of the backup to restore")
	restoreCmd.Flags().StringVarP(&restoreVersion, "version", "v", "", "Version of the backup to restore")
	restoreCmd.Flags().String("user", "", "Email to identify backup owner (optional if logged in)")
	restoreCmd.MarkFlagRequired("tag")
	restoreCmd.MarkFlagRequired("version")
}
