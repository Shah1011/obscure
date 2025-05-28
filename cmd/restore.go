package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/shah1011/obscure/internal/config"
	"github.com/shah1011/obscure/utils"
	"github.com/spf13/cobra"
)

var restoreTag string
var restoreVersion string

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore an encrypted backup from S3 or GCS",
	Run: func(cmd *cobra.Command, args []string) {
		userFlag, _ := cmd.Flags().GetString("user")
		var userID string
		var err error

		if userFlag != "" {
			userID = userFlag
		} else {
			userID, err = config.GetSessionUsername()
			if err != nil || userID == "" {
				fmt.Println("❌ You are not logged in. Use --user or run `obscure login`")
				return
			}
		}

		token, err := config.GetSessionToken()
		if err != nil || token == "" {
			fmt.Println("❌ Not logged in. Please run `obscure login` or `obscure signup`.")
			return
		}

		// Use token in Authorization header or validation

		// ✅ Get provider from session/config
		var provider string
		provider, err = config.GetSessionProvider()
		if err != nil || provider == "" {
			provider, err = config.GetUserDefaultProvider()
			if err != nil || provider == "" {
				fmt.Println("❌ No default cloud provider found for user. Please set one using `obscure switch-provider`.")
				return
			}
		}

		providerNames := map[string]string{
			"s3":  "Amazon S3",
			"gcs": "Google Cloud Storage",
		}
		providerDisplayName := providerNames[provider]
		if providerDisplayName == "" {
			providerDisplayName = provider
		}
		fmt.Printf("☁️  Using provider: %s\n", providerDisplayName)

		// 🗝️ Construct backup key (same naming convention for S3 and GCS)
		var key string
		switch provider {
		case "s3":
			key = fmt.Sprintf("backups/%s/%s/%s_backup.obscure", userID, restoreTag, restoreVersion)
		case "gcs":
			key = fmt.Sprintf("backups/%s/%s/%s_backup.obscure", userID, restoreTag, restoreVersion)
		default:
			fmt.Println("❌ Unknown provider. Supported: s3, gcs.")
			return
		}

		outputDir := fmt.Sprintf("restored_%s_v%s", restoreTag, restoreVersion)
		fmt.Println("🔍 Attempting to restore from key:", key)

		var rawReader io.ReadCloser
		var size int64

		switch provider {
		case "s3":
			size, err = utils.GetObjectSize(bucketName, key)
			if err != nil {
				if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "StatusCode: 404") {
					fmt.Printf("❌ No backup found for tag '%s' and version '%s' in S3.\n", restoreTag, restoreVersion)
				} else {
					fmt.Println("❌ Could not get backup size:", err)
				}
				return
			}

			fmt.Println("🔽 Downloading encrypted backup from S3...")
			rawReader, err = utils.DownloadFromS3Stream(bucketName, key)
			if err != nil {
				fmt.Println("❌ Failed to download backup:", err)
				return
			}

		case "gcs":
			fmt.Println("🔽 Downloading encrypted backup from GCS...")
			rawReader, size, err = utils.DownloadFromGCSStream(key)
			if err != nil {
				if strings.Contains(err.Error(), "storage: object doesn't exist") || strings.Contains(err.Error(), "Error 404") {
					fmt.Printf("❌ No backup found for tag '%s' and version '%s' in GCS.\n", restoreTag, restoreVersion)
				} else {
					fmt.Println("❌ Failed to download backup:", err)
				}
				return
			}
		default:
			fmt.Println("❌ Unknown provider.")
			return
		}
		defer rawReader.Close()

		// 🔐 Prompt for password
		password, err := utils.PromptPassword("🔐 Enter decryption password:")
		if err != nil || strings.TrimSpace(password) == "" {
			fmt.Println("❌ Invalid or empty password.")
			return
		}

		progressReader := utils.NewProgressReader(rawReader, size, "Downloading", 40)

		decStream, err := utils.DecryptStream(progressReader, password)
		if err != nil {
			fmt.Println("❌ Decryption failed:", err)
			return
		}

		err = utils.DecompressZstdToDirectory(decStream, outputDir)
		if err != nil {
			fmt.Println("❌ Failed to decompress:", err)
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
