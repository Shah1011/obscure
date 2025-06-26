package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"archive/tar"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	cfg "github.com/shah1011/obscure/internal/config"
	strg "github.com/shah1011/obscure/internal/storage"
	"github.com/shah1011/obscure/utils"
	"github.com/spf13/cobra"
)

// CreateBackupFile creates a backup file from the given path
func CreateBackupFile(path string) (*os.File, error) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "obscure-backup-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Check if path is a directory
	fileInfo, err := os.Stat(path)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	if fileInfo.IsDir() {
		// Create a tar archive for directories
		tw := tar.NewWriter(tmpFile)
		defer tw.Close()

		// Walk through the directory
		err = filepath.Walk(path, func(file string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip the root directory itself
			if file == path {
				return nil
			}

			// Create header
			header, err := tar.FileInfoHeader(fi, file)
			if err != nil {
				return err
			}

			// Get relative path
			relPath, err := filepath.Rel(path, file)
			if err != nil {
				return err
			}
			header.Name = relPath

			// Write header
			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			// If it's a regular file, write its contents
			if !fi.IsDir() {
				data, err := os.Open(file)
				if err != nil {
					return err
				}
				defer data.Close()

				if _, err := io.Copy(tw, data); err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return nil, fmt.Errorf("failed to create tar archive: %w", err)
		}

		// Close the tar writer
		if err := tw.Close(); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return nil, fmt.Errorf("failed to close tar writer: %w", err)
		}
	} else {
		// For regular files, just copy the contents
		srcFile, err := os.Open(path)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return nil, fmt.Errorf("failed to open source file: %w", err)
		}
		defer srcFile.Close()

		if _, err := io.Copy(tmpFile, srcFile); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return nil, fmt.Errorf("failed to copy file contents: %w", err)
		}
	}

	// Reset file pointer to beginning
	if _, err := tmpFile.Seek(0, 0); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, fmt.Errorf("failed to seek to beginning: %w", err)
	}

	return tmpFile, nil
}

// FormatBytes formats a byte size into a human-readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func uploadWithSpinner(ctx context.Context, reader io.Reader, size int64, uploadFn func(io.Reader) error) error {
	var wg sync.WaitGroup
	done := make(chan struct{})
	wg.Add(1)

	// Start spinner in a goroutine
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

	// Perform upload
	err := uploadFn(reader)

	// Stop spinner
	close(done)
	wg.Wait()
	fmt.Print("\r") // Clear spinner line

	return err
}

func uploadWithAWSCLI(localPath, bucket, key, region, endpoint, accessKey, secretKey string) error {
	env := os.Environ()
	env = append(env, "AWS_ACCESS_KEY_ID="+accessKey)
	env = append(env, "AWS_SECRET_ACCESS_KEY="+secretKey)

	dest := "s3://" + bucket + "/" + key
	cmd := exec.Command("aws", "--endpoint", endpoint, "s3", "cp", localPath, dest, "--region", region)
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	fmt.Print(string(output))
	if err != nil {
		return fmt.Errorf("AWS CLI upload failed: %v", err)
	}
	return nil
}

var backupCmd = &cobra.Command{
	Use:   "backup <path>",
	Short: "Back up a file or directory to your cloud storage",
	Long: `Back up a file or directory to your cloud storage. You can specify the backup tag and version using flags:
  --tag: Tag for the backup (e.g., 'unit' or 'prod')
  --version: Version for the backup (e.g., '2.1' or '1.0')
  --direct: Create an unencrypted tar backup (default is encrypted .obscure format)
  --all: Upload to all enabled cloud providers`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Get session info
		if _, err := cfg.GetSessionEmail(); err != nil {
			fmt.Println("❌ Not logged in. Please run 'obscure login' first.")
			return
		}

		// Check if direct backup is requested
		isDirect, _ := cmd.Flags().GetBool("direct")
		isAll, _ := cmd.Flags().GetBool("all")

		username, err := cfg.GetSessionUsername()
		if err != nil {
			fmt.Println("❌ Failed to get username from session:", err)
			return
		}

		// Get provider info
		providerKey, err := cfg.GetSessionProvider()
		if err != nil || providerKey == "" {
			providerKey, err = cfg.GetUserDefaultProvider()
			if err != nil || providerKey == "" {
				fmt.Println("⚠️  No cloud provider is configured.")
				return
			}
		}

		// Get provider config
		providers, err := cfg.LoadUserProviders()
		if err != nil {
			fmt.Printf("❌ Failed to load provider configuration: %v\n", err)
			return
		}

		// Get backup path and tag
		backupPath := args[0]

		// Get tag from flag or prompt
		tag, err := cmd.Flags().GetString("tag")
		if err != nil || tag == "" {
			tag, err = utils.PromptLine("🏷️  Enter a tag for this backup (e.g., 'unit' or 'prod'): ")
			if err != nil || strings.TrimSpace(tag) == "" {
				fmt.Println("❌ Invalid tag.")
				return
			}
		}

		// Get version from flag or generate
		version, err := cmd.Flags().GetString("version")
		if err != nil || version == "" {
			version = time.Now().Format("2006.01.02-15.04.05")
		}

		// Create backup
		fmt.Printf("📦 Creating backup of %s...\n", backupPath)
		start := time.Now()

		// Create backup file
		backupFile, err := CreateBackupFile(backupPath)
		if err != nil {
			fmt.Printf("❌ Failed to create backup file: %v\n", err)
			return
		}
		defer os.Remove(backupFile.Name())
		defer backupFile.Close()

		// Get file size
		fileInfo, err := backupFile.Stat()
		if err != nil {
			fmt.Printf("❌ Failed to get file info: %v\n", err)
			return
		}
		fileSize := fileInfo.Size()

		// Prepare the backup data
		var uploadReader io.Reader
		var uploadSize int64
		var extension string

		if isDirect {
			// For direct backups, use the tar file as is
			uploadReader = backupFile
			uploadSize = fileSize
			extension = "tar"
		} else {
			// For encrypted backups, prompt for password and encrypt
			fmt.Println("⚠️  WARNING: Keep your encryption password safe. If you lose it, you won't be able to recover your backup!")

			password, err := utils.PromptPassword("🔐 Enter encryption password: ")
			if err != nil || strings.TrimSpace(password) == "" {
				fmt.Println("❌ Invalid or empty password.")
				return
			}

			// Ask for password confirmation
			confirmPassword, err := utils.PromptPassword("🔐 Confirm encryption password: ")
			if err != nil || strings.TrimSpace(confirmPassword) == "" {
				fmt.Println("❌ Invalid or empty confirmation password.")
				return
			}

			if password != confirmPassword {
				fmt.Println("❌ Passwords do not match. Please try again.")
				return
			}

			// Create a buffer to store encrypted data
			var encryptedBuf bytes.Buffer

			// Create encryption writer
			encWriter, err := utils.EncryptStream(&encryptedBuf, password)
			if err != nil {
				fmt.Printf("❌ Failed to initialize encryption: %v\n", err)
				return
			}

			// Create compression writer
			compWriter := utils.NewCompressWriter(encWriter)
			defer compWriter.Close()

			// Copy data through compression and encryption
			if _, err := io.Copy(compWriter, backupFile); err != nil {
				fmt.Printf("❌ Failed to compress and encrypt: %v\n", err)
				return
			}

			// Close writers in correct order
			if err := compWriter.Close(); err != nil {
				fmt.Printf("❌ Failed to close compression: %v\n", err)
				return
			}
			if err := encWriter.Close(); err != nil {
				fmt.Printf("❌ Failed to finalize encryption: %v\n", err)
				return
			}

			uploadReader = bytes.NewReader(encryptedBuf.Bytes())
			uploadSize = int64(encryptedBuf.Len())
			extension = "obscure"
		}

		filename := fmt.Sprintf("%s_%s.%s", version, tag, extension)
		key := fmt.Sprintf("backups/%s/%s/%s", username, tag, filename)

		// Helper: upload without spinner
		uploadFnNoSpinner := func(ctx context.Context, reader io.Reader, size int64, uploadFn func(io.Reader) error) error {
			return uploadFn(reader)
		}

		uploadToProvider := func(providerKey string, config *cfg.CloudProviderConfig, suppressSpinner bool) error {
			bucket := config.Bucket
			switch providerKey {
			case "s3":
				ctx := context.Background()
				awsCfg, err := strg.NewAWSClient(ctx, "s3")
				if err != nil {
					return fmt.Errorf("failed to initialize AWS client: %v", err)
				}
				client := s3.NewFromConfig(*awsCfg)
				_, err = client.HeadObject(ctx, &s3.HeadObjectInput{
					Bucket: aws.String(bucket),
					Key:    aws.String(key),
				})
				if err == nil {
					return fmt.Errorf("a backup with this name already exists")
				}
				if suppressSpinner {
					return uploadFnNoSpinner(ctx, uploadReader, uploadSize, func(reader io.Reader) error {
						_, err := client.PutObject(ctx, &s3.PutObjectInput{
							Bucket: aws.String(bucket),
							Key:    aws.String(key),
							Body:   reader,
							Metadata: map[string]string{
								"username":  username,
								"tag":       tag,
								"version":   version,
								"is_direct": fmt.Sprintf("%v", isDirect),
							},
						})
						return err
					})
				}
				return uploadWithSpinner(ctx, uploadReader, uploadSize, func(reader io.Reader) error {
					_, err := client.PutObject(ctx, &s3.PutObjectInput{
						Bucket: aws.String(bucket),
						Key:    aws.String(key),
						Body:   reader,
						Metadata: map[string]string{
							"username":  username,
							"tag":       tag,
							"version":   version,
							"is_direct": fmt.Sprintf("%v", isDirect),
						},
					})
					return err
				})
			case "gcs":
				ctx := context.Background()
				client, err := strg.NewGCSClient(ctx, "gcs")
				if err != nil {
					return fmt.Errorf("failed to initialize GCS client: %v", err)
				}
				defer client.Close()
				_, err = client.Bucket(bucket).Object(key).Attrs(ctx)
				if err == nil {
					return fmt.Errorf("a backup with this name already exists")
				}
				if suppressSpinner {
					return uploadFnNoSpinner(ctx, uploadReader, uploadSize, func(reader io.Reader) error {
						writer := client.Bucket(bucket).Object(key).NewWriter(ctx)
						writer.Metadata = map[string]string{
							"username":  username,
							"tag":       tag,
							"version":   version,
							"is_direct": fmt.Sprintf("%v", isDirect),
						}
						if _, err := io.Copy(writer, reader); err != nil {
							return err
						}
						return writer.Close()
					})
				}
				return uploadWithSpinner(ctx, uploadReader, uploadSize, func(reader io.Reader) error {
					writer := client.Bucket(bucket).Object(key).NewWriter(ctx)
					writer.Metadata = map[string]string{
						"username":  username,
						"tag":       tag,
						"version":   version,
						"is_direct": fmt.Sprintf("%v", isDirect),
					}
					if _, err := io.Copy(writer, reader); err != nil {
						return err
					}
					return writer.Close()
				})
			case "b2":
				ctx := context.Background()
				b2Client, err := strg.NewB2Client(ctx, "b2")
				if err != nil {
					return fmt.Errorf("failed to initialize B2 client: %v", err)
				}
				exists, err := b2Client.FileExists(ctx, key)
				if err != nil {
					return fmt.Errorf("failed to check if backup exists: %v", err)
				}
				if exists {
					return fmt.Errorf("a backup with this name already exists")
				}
				if suppressSpinner {
					return uploadFnNoSpinner(ctx, uploadReader, uploadSize, func(reader io.Reader) error {
						return b2Client.UploadFile(ctx, key, reader, map[string]string{
							"username":  username,
							"tag":       tag,
							"version":   version,
							"is_direct": fmt.Sprintf("%v", isDirect),
						})
					})
				}
				return uploadWithSpinner(ctx, uploadReader, uploadSize, func(reader io.Reader) error {
					return b2Client.UploadFile(ctx, key, reader, map[string]string{
						"username":  username,
						"tag":       tag,
						"version":   version,
						"is_direct": fmt.Sprintf("%v", isDirect),
					})
				})
			case "idrive":
				ctx := context.Background()
				idriveClient, err := strg.NewIDriveClient(ctx, "idrive")
				if err != nil {
					return fmt.Errorf("failed to initialize IDrive E2 client: %v", err)
				}
				exists, err := idriveClient.FileExists(ctx, key)
				if err != nil {
					return fmt.Errorf("failed to check if backup exists: %v", err)
				}
				if exists {
					return fmt.Errorf("a backup with this name already exists")
				}
				if suppressSpinner {
					return uploadFnNoSpinner(ctx, uploadReader, uploadSize, func(reader io.Reader) error {
						return idriveClient.UploadFile(ctx, key, reader, map[string]string{
							"username":  username,
							"tag":       tag,
							"version":   version,
							"is_direct": fmt.Sprintf("%v", isDirect),
						})
					})
				}
				return uploadWithSpinner(ctx, uploadReader, uploadSize, func(reader io.Reader) error {
					return idriveClient.UploadFile(ctx, key, reader, map[string]string{
						"username":  username,
						"tag":       tag,
						"version":   version,
						"is_direct": fmt.Sprintf("%v", isDirect),
					})
				})
			case "s3-compatible":
				ctx := context.Background()
				s3CompatibleClient, err := strg.NewS3CompatibleClient(ctx, "s3-compatible")
				if err != nil {
					return fmt.Errorf("failed to initialize S3-compatible client: %v", err)
				}
				exists, err := s3CompatibleClient.FileExists(ctx, key)
				if err != nil {
					return fmt.Errorf("failed to check if backup exists: %v", err)
				}
				if exists {
					return fmt.Errorf("a backup with this name already exists")
				}
				if suppressSpinner {
					return uploadFnNoSpinner(ctx, uploadReader, uploadSize, func(reader io.Reader) error {
						return s3CompatibleClient.UploadFile(ctx, key, reader, map[string]string{
							"username":  username,
							"tag":       tag,
							"version":   version,
							"is_direct": fmt.Sprintf("%v", isDirect),
						})
					})
				}
				return uploadWithSpinner(ctx, uploadReader, uploadSize, func(reader io.Reader) error {
					return s3CompatibleClient.UploadFile(ctx, key, reader, map[string]string{
						"username":  username,
						"tag":       tag,
						"version":   version,
						"is_direct": fmt.Sprintf("%v", isDirect),
					})
				})
			case "storj":
				ctx := context.Background()
				storjClient, err := strg.NewStorjClient(ctx, "storj")
				if err != nil {
					return fmt.Errorf("failed to initialize Storj client: %v", err)
				}
				exists, err := storjClient.FileExists(ctx, key)
				if err != nil {
					return fmt.Errorf("failed to check if backup exists: %v", err)
				}
				if exists {
					return fmt.Errorf("a backup with this name already exists")
				}
				if suppressSpinner {
					return uploadFnNoSpinner(ctx, uploadReader, uploadSize, func(reader io.Reader) error {
						return storjClient.UploadFile(ctx, key, reader, map[string]string{
							"username":  username,
							"tag":       tag,
							"version":   version,
							"is_direct": fmt.Sprintf("%v", isDirect),
						})
					})
				}
				return uploadWithSpinner(ctx, uploadReader, uploadSize, func(reader io.Reader) error {
					return storjClient.UploadFile(ctx, key, reader, map[string]string{
						"username":  username,
						"tag":       tag,
						"version":   version,
						"is_direct": fmt.Sprintf("%v", isDirect),
					})
				})
			case "filebase-ipfs":
				ctx := context.Background()
				s3CompatibleClient, err := strg.NewS3CompatibleClient(ctx, "filebase-ipfs")
				if err != nil {
					return fmt.Errorf("failed to initialize Filebase+IPFS client: %v", err)
				}
				exists, err := s3CompatibleClient.FileExists(ctx, key)
				if err != nil {
					return fmt.Errorf("failed to check if backup exists: %v", err)
				}
				if exists {
					return fmt.Errorf("a backup with this name already exists")
				}
				uploadFn := func(reader io.Reader) error {
					err := s3CompatibleClient.UploadFile(ctx, key, reader, map[string]string{
						"username":  username,
						"tag":       tag,
						"version":   version,
						"is_direct": fmt.Sprintf("%v", isDirect),
					})
					if err != nil && strings.Contains(strings.ToLower(err.Error()), "access denied") {
						fmt.Print("\r\033[K")
						fmt.Println("⚠️  Go SDK upload failed to IPFS - access denied. Trying AWS CLI fallback...")
						// Save the file to a temp location for CLI upload
						tmpPath := "obscure_tmp_upload_file"
						f, ferr := os.Create(tmpPath)
						if ferr != nil {
							return fmt.Errorf("failed to create temp file for AWS CLI upload: %v", ferr)
						}
						// Prepare reader for AWS CLI upload
						var readerForCLI io.Reader = reader
						if seeker, ok := reader.(io.Seeker); ok {
							seeker.Seek(0, io.SeekStart)
						} else if buf, ok := reader.(*bytes.Buffer); ok {
							readerForCLI = bytes.NewReader(buf.Bytes())
						}
						_, ferr = io.Copy(f, readerForCLI)
						f.Close()
						if ferr != nil {
							os.Remove(tmpPath)
							return fmt.Errorf("failed to write temp file for AWS CLI upload: %v", ferr)
						}
						providerConfig, _ := cfg.GetProviderConfig("filebase-ipfs")
						err = uploadWithAWSCLI(tmpPath, providerConfig.Bucket, key, providerConfig.Region, providerConfig.FilebaseEndpoint, providerConfig.AccessKeyID, providerConfig.SecretAccessKey)
						os.Remove(tmpPath)
						if err != nil {
							return fmt.Errorf("AWS CLI upload failed: %v", err)
						}
						fmt.Println("✅ Backup uploaded using AWS CLI fallback.")
						return nil
					}
					return err
				}
				if suppressSpinner {
					return uploadFnNoSpinner(ctx, uploadReader, uploadSize, uploadFn)
				}
				return uploadWithSpinner(ctx, uploadReader, uploadSize, uploadFn)
			default:
				return fmt.Errorf("unsupported provider: %s", providerKey)
			}
		}

		supportedProviders := map[string]bool{
			"s3":            true,
			"gcs":           true,
			"b2":            true,
			"idrive":        true,
			"s3-compatible": true,
			"storj":         true,
			"filebase-ipfs": true,
		}

		if isAll {
			results := make(map[string]error)
			var providerList []string
			var configList []*cfg.CloudProviderConfig
			for key, config := range providers.Providers {
				if !supportedProviders[key] || !config.Enabled {
					continue
				}
				isComplete, _ := cfg.IsProviderConfigComplete(config)
				if !isComplete {
					continue
				}
				providerList = append(providerList, key)
				configList = append(configList, config)
			}
			if len(providerList) == 0 {
				fmt.Println("❌ No enabled and fully configured providers found.")
				return
			}
			// Single spinner for all uploads
			var wg sync.WaitGroup
			done := make(chan struct{})
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
							fmt.Printf("\r☁️ Uploading to all enabled cloud providers... %s", string(r))
							time.Sleep(100 * time.Millisecond)
						}
					}
				}
			}()
			// Perform uploads (sequentially)
			for i, key := range providerList {
				results[key] = uploadToProvider(key, configList[i], true)
			}
			close(done)
			wg.Wait()
			fmt.Print("\r\033[K") // Clear spinner line
			fmt.Println("\n📊 Upload results:")
			for key, err := range results {
				if err == nil {
					fmt.Printf("✅ %s: Success\n", strings.ToUpper(key))
				} else {
					fmt.Printf("❌ %s: %v\n", strings.ToUpper(key), err)
				}
			}
			elapsed := time.Since(start)
			fmt.Printf("\n✅ Backup completed in %s\n", elapsed.Round(time.Millisecond))
			fmt.Printf("📊 File size: %s\n", FormatBytes(uploadSize))
			return
		}

		// Default: upload to single provider
		config, ok := providers.Providers[providerKey]
		if !ok || !config.Enabled {
			fmt.Printf("❌ Provider %s is not configured or disabled\n", strings.ToUpper(providerKey))
			return
		}
		if err := uploadToProvider(providerKey, config, false); err != nil {
			fmt.Printf("❌ Failed to upload: %v\n", err)
			return
		}
		elapsed := time.Since(start)
		fmt.Printf("✅ Backup completed in %s\n", elapsed.Round(time.Millisecond))
		fmt.Printf("📊 File size: %s\n", FormatBytes(uploadSize))
		fmt.Printf("🔗 Backup path: %s\n", key)
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.Flags().StringP("tag", "t", "", "Tag for the backup (e.g., 'unit' or 'prod')")
	backupCmd.Flags().StringP("version", "v", "", "Version for the backup (e.g., '2.1' or '1.0')")
	backupCmd.Flags().BoolP("direct", "d", false, "Create an unencrypted tar backup (default is encrypted .obscure format)")
	backupCmd.Flags().BoolP("all", "a", false, "Upload to all enabled cloud providers")
}
