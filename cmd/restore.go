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
var isDirectRestore bool

var restoreCmd = &cobra.Command{
	Use:   "restore [backup_path]",
	Short: "Restore a backup from S3 or GCS",
	Long: `Restore a backup from S3 or GCS. You can specify the backup in two ways:
1. Using flags: --tag and --version
2. Using path format: tag/version_backup.extension (e.g., testdata/2.1_backup.tar)`,
	Args: func(cmd *cobra.Command, args []string) error {
		// If no args provided, require tag and version flags
		if len(args) == 0 {
			if restoreTag == "" || restoreVersion == "" {
				return fmt.Errorf("either provide a backup path or use --tag and --version flags")
			}
			return nil
		}

		// Parse backup path if provided
		if len(args) == 1 {
			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid backup path format. Use: tag/version_backup.extension")
			}

			// Extract tag
			restoreTag = parts[0]

			// Extract version and check extension
			versionParts := strings.Split(parts[1], "_backup.")
			if len(versionParts) != 2 {
				return fmt.Errorf("invalid backup path format. Use: tag/version_backup.extension")
			}
			restoreVersion = versionParts[0]

			// Check if it's a direct backup
			extension := versionParts[1]
			if extension == "tar" {
				isDirectRestore = true
			} else if extension != "obscure" {
				return fmt.Errorf("unsupported backup extension: %s. Use .obscure or .tar", extension)
			}
			return nil
		}

		return fmt.Errorf("too many arguments")
	},
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

		// Get provider from session/config
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

		// Construct backup key with correct extension
		extension := "obscure"
		if isDirectRestore {
			extension = "tar"
		}
		key := fmt.Sprintf("backups/%s/%s/%s_backup.%s", userID, restoreTag, restoreVersion, extension)
		fmt.Println("🔍 Attempting to restore from key:", key)

		outputDir := fmt.Sprintf("restored_%s_v%s", restoreTag, restoreVersion)

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

			fmt.Println("🔽 Downloading backup from S3...")
			rawReader, err = utils.DownloadFromS3Stream(bucketName, key)
			if err != nil {
				fmt.Println("❌ Failed to download backup:", err)
				return
			}

		case "gcs":
			fmt.Println("🔽 Downloading backup from GCS...")
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

		progressReader := utils.NewProgressReader(rawReader, size, "🔽 Downloading", 40)

		if isDirectRestore {
			// For direct backups, just extract the tar archive
			err = utils.ExtractTarArchive(progressReader, outputDir)
			if err != nil {
				fmt.Println("❌ Failed to extract tar archive:", err)
				return
			}
		} else {
			// For encrypted backups, decrypt and decompress
			password, err := utils.PromptPassword("🔐 Enter decryption password:")
			if err != nil || strings.TrimSpace(password) == "" {
				fmt.Println("❌ Invalid or empty password.")
				return
			}

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
		}

		fmt.Println("\n✅ Restore complete at:", outputDir)
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	restoreCmd.Flags().StringVarP(&restoreTag, "tag", "t", "", "Tag of the backup to restore")
	restoreCmd.Flags().StringVarP(&restoreVersion, "version", "v", "", "Version of the backup to restore")
	restoreCmd.Flags().String("user", "", "Email to identify backup owner (optional if logged in)")
}
