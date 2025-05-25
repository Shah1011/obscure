package cmd

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/shah1011/obscure/internal/auth"
	"github.com/shah1011/obscure/internal/config"
	"github.com/shah1011/obscure/utils"
	"github.com/spf13/cobra"
)

var tag string
var version string
var bucketName = "obscure-open" // Change this if needed

var backupCmd = &cobra.Command{
	Use:   "backup [directory]",
	Args:  cobra.ExactArgs(1),
	Short: "Back up and encrypt a directory, then upload to selected cloud",
	Run: func(cmd *cobra.Command, args []string) {
		inputDir := args[0]

		userFlag, _ := cmd.Flags().GetString("user")
		var userID string
		if userFlag != "" {
			userID = userFlag
		} else {
			var err error
			userID, err = config.GetSessionEmail()
			if err != nil || userID == "" {
				fmt.Println("❌ You are not logged in. Use --user or run `obscure login`")
				return
			}
		}

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

		// 🌩️ Prompt cloud provider
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

		// Upload based on cloud
		username, err := auth.GetUsernameByEmail(userID)
		if err != nil {
			fmt.Println("❌ Failed to get username:", err)
			return
		}

		done := make(chan bool)
		var wg sync.WaitGroup
		wg.Add(1)

		// Start spinner in goroutine
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

		// Key used internally for existence check (no _backup suffix)
		keyPath := fmt.Sprintf("backups/%s/%s/%s_backup.obscure", username, tag, version)

		switch idx {
		case 0: // S3
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

			// Start timing upload process
			startTime := time.Now()

			// Upload first: this prints backend response internally
			err = utils.UploadToS3Backend(
				encryptedData.Bytes(),
				username,
				tag,
				version,
				"http://localhost:8080/s3-upload",
			)
			done <- true // Signal spinner to stop
			wg.Wait()    // Wait for spinner to finish
			fmt.Print("\r\033[K")
			if err != nil {
				fmt.Printf("❌ S3 Upload via backend failed: %v\n", err)
				return
			}

			// Print file size and elapsed time in the same line as in your example
			elapsed := time.Since(startTime)
			sizeMB := float64(encryptedData.Len()) / (1024 * 1024)

			// Print file size and time taken on the same line as progress bar finished (new line)
			fmt.Printf("📦 File size: %.2f MB | ⏱ Time taken: %.2fs\n", sizeMB, elapsed.Seconds())

		case 1: // GCS
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
				encryptedData.Bytes(),
				username,
				tag,
				version,
				"http://localhost:8080/gcs-upload",
			)

			done <- true // Stop spinner
			wg.Wait()
			fmt.Print("\r\033[K") // Clear spinner line

			if err != nil {
				fmt.Printf("❌ GCS Upload failed: %v\n", err)
				return
			}

			// Same UX as S3
			elapsed := time.Since(startTime)
			sizeMB := float64(encryptedData.Len()) / (1024 * 1024)

			fmt.Printf("✅ Uploaded to GCS: backups/%s/%s/%s_backup.obscure\n", username, tag, version)
			fmt.Printf("📦 File size: %.2f MB | ⏱ Time taken: %.2fs\n", sizeMB, elapsed.Seconds())
		}

		fmt.Println("🎉 Backup completed successfully!")
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.Flags().StringVarP(&tag, "tag", "t", "", "Tag for the backup")
	backupCmd.Flags().StringVarP(&version, "version", "v", "", "Version for the backup")
	backupCmd.Flags().String("user", "", "Email to identify backup owner (optional if logged in)")
	backupCmd.MarkFlagRequired("tag")
	backupCmd.MarkFlagRequired("version")
}
