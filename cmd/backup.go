package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/shah1011/obscure/internal/config"
	"github.com/shah1011/obscure/utils"

	"github.com/spf13/cobra"
)

var tag string
var version string
var bucketName = "obscure-open"
var directUpload bool

var backupCmd = &cobra.Command{
	Use:   "backup [directory]",
	Args:  cobra.ExactArgs(1),
	Short: "Back up and encrypt a directory, then upload to selected cloud",
	Run: func(cmd *cobra.Command, args []string) {
		inputDir := args[0]

		// Check if user is logged in
		_, err := config.GetSessionEmail()
		if err != nil {
			fmt.Println("❌ You are not logged in. Please run `obscure login`")
			return
		}

		token, err := config.GetSessionToken()
		if err != nil || token == "" {
			fmt.Println("❌ Could not read auth token. Please log in first using `obscure login`.")
			return
		}

		var dataToUpload []byte
		if !directUpload {
			fmt.Println("⚠️ IMPORTANT: Save this password in a secure location. You will need it to restore the backup or else the backup will be lost forever!")
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

			fmt.Println("🔹 Compressing directory:", inputDir)
			zipBuffer, err := utils.CompressDirectoryToZstd(inputDir)
			if err != nil {
				fmt.Println("❌ Failed to zip directory:", err)
				return
			}
			fmt.Println("✅ Compressed in-memory.")

			fmt.Println("🔹 Encrypting data...")
			encryptedData, err := utils.EncryptBuffer(zipBuffer, password)
			if err != nil {
				fmt.Println("❌ Failed to encrypt data:", err)
				return
			}
			fmt.Println("✅ Data encrypted in-memory.")
			dataToUpload = encryptedData.Bytes()
		} else {
			// Direct upload - just read the directory as is
			fmt.Println("🔹 Reading directory for direct upload:", inputDir)
			dirData, err := utils.ReadDirectoryAsBytes(inputDir)
			if err != nil {
				fmt.Println("❌ Failed to read directory:", err)
				return
			}
			dataToUpload = dirData
		}

		// ✅ Fixed: Try to get provider from session first, then fallback to user default
		var provider string

		// First try to get the session provider (current active provider)
		provider, err = config.GetSessionProvider()
		if err != nil || provider == "" {
			// Fallback to user default provider
			provider, err = config.GetUserDefaultProvider()
			if err != nil || provider == "" {
				fmt.Println("❌ No default cloud provider found for user. Please set one using `obscure switch-provider`.")
				return
			}
		}

		// Map provider keys to friendly names
		providerNames := map[string]string{
			"s3":  "Amazon S3",
			"gcs": "Google Cloud Storage",
		}

		providerDisplayName := providerNames[provider]
		if providerDisplayName == "" {
			providerDisplayName = provider // fallback to original if not found
		}

		fmt.Printf("☁️  Using provider: %s\n", providerDisplayName)

		username, err := config.GetSessionUsername()
		if err != nil || username == "" {
			fmt.Println("❌ Failed to get username from session. Please log in again.")
			return
		}

		done := make(chan bool)
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			spinnerRunes := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					for _, r := range spinnerRunes {
						fmt.Printf("\r🔹 Uploading to cloud... %s", string(r))
						time.Sleep(100 * time.Millisecond)
					}
				}
			}
		}()

		keyPath := fmt.Sprintf("backups/%s/%s/%s_backup.obscure", username, tag, version)

		switch provider {
		case "s3":
			exists, err := utils.CheckIfS3ObjectExists(bucketName, keyPath)
			if err != nil {
				done <- true
				wg.Wait()
				fmt.Printf("\n❌ Failed to check existing backups: %v\n", err)
				return
			}
			if exists {
				done <- true
				wg.Wait()
				fmt.Printf("\n❌ A backup with tag '%s' and version '%s' already exists in S3.\n", tag, version)
				return
			}

			startTime := time.Now()

			err = utils.UploadToS3Backend(
				dataToUpload,
				username,
				tag,
				version,
				"http://localhost:8080/s3-upload",
				token,
				directUpload,
			)
			done <- true
			wg.Wait()
			fmt.Print("\r\033[K")
			if err != nil {
				if strings.Contains(err.Error(), "session expired") {
					fmt.Println("❌", err.Error())
					return
				}
				fmt.Printf("❌ S3 Upload via backend failed: %v\n", err)
				return
			}

			elapsed := time.Since(startTime)
			sizeMB := float64(len(dataToUpload)) / (1024 * 1024)
			extension := "obscure"
			if directUpload {
				extension = "tar"
			}
			fmt.Printf("✅ Uploaded to S3: backups/%s/%s/%s_backup.%s\n", username, tag, version, extension)
			fmt.Printf("📦 File size: %.2f MB | ⏱ Time taken: %.2fs\n", sizeMB, elapsed.Seconds())

		case "gcs":
			exists, err := utils.CheckIfGCSObjectExists(bucketName, keyPath)
			if err != nil {
				done <- true
				wg.Wait()
				fmt.Printf("❌ Failed to check GCS: %v\n", err)
				return
			}
			if exists {
				done <- true
				wg.Wait()
				fmt.Printf("❌ A backup with tag '%s' and version '%s' already exists in GCS.\n", tag, version)
				return
			}

			startTime := time.Now()

			err = utils.UploadToGCSBackend(
				dataToUpload,
				username,
				tag,
				version,
				"http://localhost:8080/gcs-upload",
				token,
				directUpload,
			)

			done <- true
			wg.Wait()
			fmt.Print("\r\033[K")
			if err != nil {
				if strings.Contains(err.Error(), "session expired") {
					fmt.Println("❌", err.Error())
					return
				}
				fmt.Fprintln(os.Stderr, err)
				return
			}

			elapsed := time.Since(startTime)
			sizeMB := float64(len(dataToUpload)) / (1024 * 1024)
			extension := "obscure"
			if directUpload {
				extension = "tar"
			}
			fmt.Printf("✅ Uploaded to GCS: backups/%s/%s/%s_backup.%s\n", username, tag, version, extension)
			fmt.Printf("📦 File size: %.2f MB | ⏱ Time taken: %.2fs\n", sizeMB, elapsed.Seconds())

		default:
			done <- true
			wg.Wait()
			fmt.Println("❌ Unknown provider. Supported: s3, gcs.")
			return
		}

		fmt.Println("🎉 Backup completed successfully!")
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.Flags().StringVarP(&tag, "tag", "t", "", "Tag for the backup")
	backupCmd.Flags().StringVarP(&version, "version", "v", "", "Version for the backup")
	backupCmd.Flags().String("user", "", "Email to identify backup owner (optional if logged in)")
	backupCmd.Flags().BoolVarP(&directUpload, "direct", "d", false, "Direct upload without encryption or compression")
	backupCmd.MarkFlagRequired("tag")
	backupCmd.MarkFlagRequired("version")
}
