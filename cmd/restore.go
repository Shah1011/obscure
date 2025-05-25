package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
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

		// ☁️ Prompt user to select provider
		providers := []string{"Amazon S3", "Google Cloud Storage"}
		underline := "\033[4m"
		reset := "\033[0m"
		prompt := promptui.Select{
			Label: "Select Cloud Provider",
			Items: providers,
			Templates: &promptui.SelectTemplates{
				Label:    "{{ . }}",
				Active:   underline + "{{ . | green }}" + reset,
				Inactive: "{{ . }}",
				Selected: "☁️  Selected: {{ . | green }}",
			},
			Stdout: os.Stderr,
		}
		idx, _, err := prompt.Run()
		if err != nil {
			fmt.Println("❌ Cloud selection failed:", err)
			return
		}

		// 🧬 Construct storage key based on provider
		var key string
		switch idx {
		case 0: // Amazon S3
			key = fmt.Sprintf("backups/%s/%s/%s_backup.obscure", userID, restoreTag, restoreVersion)
		case 1: // Google Cloud Storage
			key = fmt.Sprintf("%s/%s/%s_%s_v%s.obscure", userID, restoreTag, restoreVersion, restoreTag, restoreVersion)
		default:
			fmt.Println("❌ Unknown cloud provider selected.")
			return
		}

		outputDir := fmt.Sprintf("restored_%s_v%s", restoreTag, restoreVersion)
		fmt.Println("🔍 Attempting to restore from key:", key)

		// 🔐 Prompt for decryption password
		password, err := utils.PromptPassword("🔐 Enter decryption password:")
		if err != nil || strings.TrimSpace(password) == "" {
			fmt.Println("❌ Invalid or empty password.")
			return
		}

		switch idx {
		case 0: // Amazon S3
			size, err := utils.GetObjectSize(bucketName, key)
			if err != nil {
				if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "StatusCode: 404") {
					fmt.Printf("❌ No backup found for tag '%s' and version '%s' in S3.\n", restoreTag, restoreVersion)
				} else {
					fmt.Println("❌ Could not get backup size:", err)
				}
				return
			}

			fmt.Println("🔽 Downloading encrypted backup from S3...")
			rawReader, err := utils.DownloadFromS3Stream(bucketName, key)
			if err != nil {
				fmt.Println("❌ Failed to download backup:", err)
				return
			}
			defer rawReader.Close()

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

		case 1: // Google Cloud Storage
			fmt.Println("🔽 Downloading encrypted backup from GCS...")
			rawReader, size, err := utils.DownloadFromGCSStream(key)
			if err != nil {
				if strings.Contains(err.Error(), "storage: object doesn't exist") || strings.Contains(err.Error(), "Error 404") {
					fmt.Printf("❌ No backup found for tag '%s' and version '%s' in GCS.\n", restoreTag, restoreVersion)
				} else {
					fmt.Println("❌ Failed to download backup from GCS:", err)
				}
				return
			}
			defer rawReader.Close()

			progressReader := utils.NewProgressReader(rawReader, size, "Downloading", 40)

			decStream, err := utils.DecryptStream(progressReader, password)
			if err != nil {
				fmt.Println("❌ Decryption failed:", err)
				return
			}

			err = utils.DecompressZstdToDirectory(decStream, outputDir)
			if err != nil {
				fmt.Println("❌ Failed to unzip:", err)
				return
			}
		default:
			fmt.Println("❌ Unknown cloud provider selected.")
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
