package cmd

import (
	"fmt"
	"io"
	"strings"

	cfg "github.com/shah1011/obscure/internal/config"
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
   Example: obscure restore --tag=testdata --version=2.9
2. Using path format: tag/version_tag.obscure
   Example: obscure restore testdata/2.9_testdata.obscure

You can also combine both formats, but the flags will take precedence.`,
	Args: func(cmd *cobra.Command, args []string) error {

		// If no args provided and no flags, show error
		if len(args) == 0 {
			return fmt.Errorf("either provide a backup path or use --tag and --version flags")
		}

		// Parse backup path if provided
		if len(args) == 1 {
			path := args[0]

			// Check if path contains a slash (tag/path format)
			if strings.Contains(path, "/") {
				parts := strings.Split(path, "/")

				if len(parts) != 2 {
					return fmt.Errorf("invalid path format. Expected: tag/version_tag.obscure")
				}

				// Only set tag if flag wasn't provided
				if restoreTag == "" {
					restoreTag = parts[0]
				}
				path = parts[1]
			}

			// Find the last dot to handle version numbers with dots
			lastDotIndex := strings.LastIndex(path, ".")
			if lastDotIndex == -1 {
				return fmt.Errorf("invalid filename format. Expected: version_tag.obscure")
			}

			// Split into name and extension
			name := path[:lastDotIndex]
			extension := path[lastDotIndex+1:]

			// Extract version and tag from filename (e.g., "2.9_testdata")
			versionTag := strings.Split(name, "_")

			if len(versionTag) != 2 {
				return fmt.Errorf("invalid filename format. Expected: version_tag.obscure")
			}

			// Only set version if flag wasn't provided
			if restoreVersion == "" {
				restoreVersion = versionTag[0]
			}

			// Only set tag if flag wasn't provided and we didn't get it from the path
			if restoreTag == "" {
				restoreTag = versionTag[1]
			}

			// Check extension
			if extension == "tar" {
				isDirectRestore = true
			} else if extension != "obscure" {
				return fmt.Errorf("unsupported backup extension: %s. Use .obscure or .tar", extension)
			}

			// Validate that we have both tag and version
			if restoreTag == "" || restoreVersion == "" {
				return fmt.Errorf("could not determine tag or version from path. Use --tag and --version flags")
			}

			return nil
		}

		return fmt.Errorf("too many arguments")
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Validate that we have both tag and version
		if restoreTag == "" || restoreVersion == "" {
			fmt.Println("❌ Both tag and version are required. Use --tag and --version flags or provide a backup path.")
			return
		}

		userFlag, _ := cmd.Flags().GetString("user")
		var userID string
		var err error

		if userFlag != "" {
			userID = userFlag
		} else {
			userID, err = cfg.GetSessionUsername()
			if err != nil || userID == "" {
				fmt.Println("❌ You are not logged in. Use --user or run `obscure login`")
				return
			}
		}

		token, err := cfg.GetSessionToken()
		if err != nil || token == "" {
			fmt.Println("❌ Not logged in. Please run `obscure login` or `obscure signup`.")
			return
		}

		// Get provider from session/config
		var provider string
		provider, err = cfg.GetSessionProvider()
		if err != nil || provider == "" {
			provider, err = cfg.GetUserDefaultProvider()
			if err != nil || provider == "" {
				fmt.Println("❌ No default cloud provider found for user. Please set one using `obscure switch-provider`.")
				return
			}
		}

		// Get provider config
		providers, err := cfg.LoadUserProviders()
		if err != nil {
			fmt.Printf("❌ Failed to load provider configuration: %v\n", err)
			return
		}

		config, ok := providers.Providers[provider]
		if !ok || !config.Enabled {
			fmt.Printf("❌ Provider %s is not configured or disabled\n", strings.ToUpper(provider))
			return
		}

		bucket := config.Bucket

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
		key := fmt.Sprintf("backups/%s/%s/%s_%s.%s", userID, restoreTag, restoreVersion, restoreTag, extension)
		fmt.Println("🔍 Attempting to restore from key:", key)

		outputDir := fmt.Sprintf("restored_%s_v%s", restoreTag, restoreVersion)

		var rawReader io.ReadCloser
		var size int64

		switch provider {
		case "s3":
			size, err = utils.GetObjectSize(bucket, key)
			if err != nil {
				if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "StatusCode: 404") {
					fmt.Printf("❌ No backup found for tag '%s' and version '%s' in S3.\n", restoreTag, restoreVersion)
				} else {
					fmt.Println("❌ Could not get backup size:", err)
				}
				return
			}

			fmt.Println("🔽 Downloading backup from S3...")
			rawReader, err = utils.DownloadFromS3Stream(bucket, key)
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
