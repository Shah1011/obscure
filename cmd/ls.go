package cmd

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fatih/color"
	cfg "github.com/shah1011/obscure/internal/config"
	strg "github.com/shah1011/obscure/internal/storage"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all available backups (tags and versions)",
	Run: func(cmd *cobra.Command, args []string) {
		providerKey, err := cfg.GetSessionProvider()
		if err != nil || providerKey == "" {
			providerKey, err = cfg.GetUserDefaultProvider()
			if err != nil || providerKey == "" {
				fmt.Println("⚠️  No cloud provider is configured.")
				return
			}
		}

		username, _ := cfg.GetSessionUsername()

		token, err := cfg.GetSessionToken()
		if err != nil || token == "" {
			fmt.Println("❌ Not logged in. Please run `obscure login` or `obscure signup`.")
			return
		}

		// Use token in Authorization header or validation

		switch providerKey {
		case "gcs":
			prefix := fmt.Sprintf("backups/%s/", username) // new
			listFromGCS(prefix)
		case "s3":
			prefix := fmt.Sprintf("backups/%s/", username) // e.g., "backups/abul/"
			listFromS3(prefix)
		case "b2":
			prefix := fmt.Sprintf("backups/%s/", username)
			listFromB2(prefix)
		case "idrive":
			prefix := fmt.Sprintf("backups/%s/", username)
			listFromIDrive(prefix)
		default:
			fmt.Printf("❌ Unknown provider: %s\n", providerKey)
		}
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)
}

func listFromGCS(prefix string) {
	ctx := context.Background()
	client, err := strg.NewGCSClient(ctx, "gcs")
	if err != nil {
		// Check if it's a configuration error and provide helpful guidance
		if strings.Contains(err.Error(), "configuration incomplete") {
			fmt.Println("❌ GCS provider is not properly configured.")
			fmt.Println("   Missing required configuration fields.")
			fmt.Println("   Run: ./obscure provider add gcs")
			fmt.Println("   Or check configuration with: ./obscure provider list")
			return
		}
		if strings.Contains(err.Error(), "not configured") {
			fmt.Println("❌ GCS provider is not configured.")
			fmt.Println("   Run: ./obscure provider add gcs")
			return
		}
		if strings.Contains(err.Error(), "disabled") {
			fmt.Println("❌ GCS provider is disabled.")
			fmt.Println("   Complete the configuration to enable it.")
			fmt.Println("   Run: ./obscure provider add gcs")
			return
		}
		fmt.Println("❌ Error initializing GCS client:", err)
		return
	}
	defer client.Close()

	// Get bucket name from provider config
	bucket, err := strg.GetBucketName("gcs")
	if err != nil {
		fmt.Printf("❌ Failed to get bucket name: %v\n", err)
		return
	}

	it := client.Bucket(bucket).Objects(ctx, &storage.Query{Prefix: prefix})
	files := []string{}
	metadata := make(map[string]bool) // Store is_direct metadata for each file

	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			// Comprehensive error handling for GCS-specific issues
			errMsg := err.Error()

			if strings.Contains(errMsg, "invalid_grant") {
				fmt.Println("❌ Invalid GCS service account credentials.")
				fmt.Println("   Check your service account JSON file.")
				fmt.Println("   Run: ./obscure provider add gcs to update credentials")
				return
			}

			if strings.Contains(errMsg, "storage: bucket doesn't exist") {
				fmt.Println("❌ GCS bucket not found.")
				fmt.Println("   Check your bucket name in the Google Cloud console.")
				fmt.Println("   Run: ./obscure provider add gcs to update bucket name")
				return
			}

			if strings.Contains(errMsg, "storage: permission denied") {
				fmt.Println("❌ Access denied to GCS bucket.")
				fmt.Println("   Possible issues:")
				fmt.Println("   - Incorrect service account permissions")
				fmt.Println("   - Bucket doesn't exist")
				fmt.Println("   - Service account doesn't have access to this bucket")
				fmt.Println("   Check your service account permissions in the Google Cloud console.")
				return
			}

			if strings.Contains(errMsg, "exceeded maximum number of attempts") {
				fmt.Println("❌ GCS connection timeout.")
				fmt.Println("   Possible issues:")
				fmt.Println("   - Network connectivity problem")
				fmt.Println("   - Google Cloud service is slow or down")
				fmt.Println("   - Incorrect project configuration")
				fmt.Println("   Run: ./obscure provider add gcs to check configuration")
				return
			}

			if strings.Contains(errMsg, "invalid character") || strings.Contains(errMsg, "unexpected end of JSON") {
				fmt.Println("❌ Invalid GCS service account JSON file.")
				fmt.Println("   The JSON file appears to be corrupted or invalid.")
				fmt.Println("   Download a fresh service account key from Google Cloud console.")
				fmt.Println("   Run: ./obscure provider add gcs to update credentials")
				return
			}

			// Generic error with suggestion to check configuration
			fmt.Println("❌ Failed to list GCS backups:", err)
			fmt.Println("   This might be due to:")
			fmt.Println("   - Invalid service account credentials")
			fmt.Println("   - Incorrect bucket name")
			fmt.Println("   - Network connectivity issues")
			fmt.Println("   Run: ./obscure provider add gcs to reconfigure")
			return
		}
		files = append(files, obj.Name)
		// Check if this is a direct backup
		isDirect := obj.Metadata != nil && obj.Metadata["is_direct"] == "true"
		metadata[obj.Name] = isDirect
	}

	printBackups(files, metadata)
}

func listFromS3(prefix string) {
	ctx := context.Background()
	awsCfg, err := strg.NewAWSClient(ctx, "s3")
	if err != nil {
		// Check if it's a configuration error and provide helpful guidance
		if strings.Contains(err.Error(), "configuration incomplete") {
			fmt.Println("❌ S3 provider is not properly configured.")
			fmt.Println("   Missing required configuration fields.")
			fmt.Println("   Run: ./obscure provider add s3")
			fmt.Println("   Or check configuration with: ./obscure provider list")
			return
		}
		if strings.Contains(err.Error(), "not configured") {
			fmt.Println("❌ S3 provider is not configured.")
			fmt.Println("   Run: ./obscure provider add s3")
			return
		}
		if strings.Contains(err.Error(), "disabled") {
			fmt.Println("❌ S3 provider is disabled.")
			fmt.Println("   Complete the configuration to enable it.")
			fmt.Println("   Run: ./obscure provider add s3")
			return
		}
		fmt.Println("❌ Failed to load AWS config:", err)
		return
	}

	client := s3sdk.NewFromConfig(*awsCfg)

	// Get bucket name from provider config
	bucket, err := strg.GetBucketName("s3")
	if err != nil {
		fmt.Printf("❌ Failed to get bucket name: %v\n", err)
		return
	}

	input := &s3sdk.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	paginator := s3sdk.NewListObjectsV2Paginator(client, input)
	files := []string{}
	metadata := make(map[string]bool) // Store is_direct metadata for each file

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			// Comprehensive error handling for S3-specific issues
			errMsg := err.Error()

			if strings.Contains(errMsg, "InvalidAccessKeyId") {
				fmt.Println("❌ Invalid AWS Access Key ID.")
				fmt.Println("   Check your AWS Access Key ID in the AWS console.")
				fmt.Println("   Run: ./obscure provider add s3 to update credentials")
				return
			}

			if strings.Contains(errMsg, "SignatureDoesNotMatch") {
				fmt.Println("❌ Invalid AWS Secret Access Key.")
				fmt.Println("   Check your AWS Secret Access Key in the AWS console.")
				fmt.Println("   Run: ./obscure provider add s3 to update credentials")
				return
			}

			if strings.Contains(errMsg, "NoSuchBucket") {
				fmt.Println("❌ S3 bucket not found.")
				fmt.Println("   Check your bucket name in the AWS console.")
				fmt.Println("   Run: ./obscure provider add s3 to update bucket name")
				return
			}

			if strings.Contains(errMsg, "AccessDenied") {
				fmt.Println("❌ Access denied to S3 bucket.")
				fmt.Println("   Possible issues:")
				fmt.Println("   - Incorrect IAM permissions")
				fmt.Println("   - Bucket doesn't exist")
				fmt.Println("   - IAM user doesn't have access to this bucket")
				fmt.Println("   Check your IAM user permissions in the AWS console.")
				return
			}

			if strings.Contains(errMsg, "exceeded maximum number of attempts") {
				fmt.Println("❌ S3 connection timeout.")
				fmt.Println("   Possible issues:")
				fmt.Println("   - Incorrect region")
				fmt.Println("   - Network connectivity problem")
				fmt.Println("   - AWS service is slow or down")
				fmt.Println("   Run: ./obscure provider add s3 to check/update region")
				return
			}

			// Generic error with suggestion to check configuration
			fmt.Println("❌ Failed to list S3 backups:", err)
			fmt.Println("   This might be due to:")
			fmt.Println("   - Incorrect region")
			fmt.Println("   - Invalid credentials")
			fmt.Println("   - Network connectivity issues")
			fmt.Println("   Run: ./obscure provider add s3 to reconfigure")
			return
		}
		for _, obj := range page.Contents {
			// Get object metadata to check if it's a direct backup
			headInput := &s3sdk.HeadObjectInput{
				Bucket: aws.String(bucket),
				Key:    obj.Key,
			}
			headOutput, err := client.HeadObject(ctx, headInput)
			if err != nil {
				continue
			}
			files = append(files, *obj.Key)
			if headOutput.Metadata != nil {
				isDirect := headOutput.Metadata["is_direct"] == "true"
				metadata[*obj.Key] = isDirect
			}
		}
	}

	printBackups(files, metadata)
}

func listFromB2(prefix string) {
	ctx := context.Background()
	b2Client, err := strg.NewB2Client(ctx, "b2")
	if err != nil {
		// Check if it's a configuration error and provide helpful guidance
		if strings.Contains(err.Error(), "configuration incomplete") {
			fmt.Println("❌ B2 provider is not properly configured.")
			fmt.Println("   Missing required configuration fields.")
			fmt.Println("   Run: ./obscure provider add b2")
			fmt.Println("   Or check configuration with: ./obscure provider list")
			return
		}
		if strings.Contains(err.Error(), "not configured") {
			fmt.Println("❌ B2 provider is not configured.")
			fmt.Println("   Run: ./obscure provider add b2")
			return
		}
		if strings.Contains(err.Error(), "disabled") {
			fmt.Println("❌ B2 provider is disabled.")
			fmt.Println("   Complete the configuration to enable it.")
			fmt.Println("   Run: ./obscure provider add b2")
			return
		}
		fmt.Println("❌ Failed to load B2 config:", err)
		return
	}

	// List files using official B2 SDK
	files, err := b2Client.ListFiles(ctx, prefix)
	if err != nil {
		// Comprehensive error handling for B2-specific issues
		errMsg := err.Error()

		// Check for endpoint/URL issues
		if strings.Contains(errMsg, "tls: failed to verify certificate") {
			fmt.Println("❌ B2 endpoint certificate verification failed.")
			fmt.Println("   This usually means the endpoint URL is incorrect.")
			fmt.Println("   Check your B2 bucket's endpoint in the Backblaze console.")
			fmt.Println("   Run: ./obscure provider add b2 to update the endpoint")
			return
		}

		if strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "dial tcp") {
			fmt.Println("❌ Cannot connect to B2 endpoint.")
			fmt.Println("   Possible issues:")
			fmt.Println("   - Incorrect endpoint URL")
			fmt.Println("   - Network connectivity problem")
			fmt.Println("   - B2 service is down")
			fmt.Println("   Run: ./obscure provider add b2 to check/update endpoint")
			return
		}

		if strings.Contains(errMsg, "exceeded maximum number of attempts") {
			fmt.Println("❌ B2 connection timeout.")
			fmt.Println("   Possible issues:")
			fmt.Println("   - Incorrect endpoint URL")
			fmt.Println("   - Network connectivity problem")
			fmt.Println("   - B2 service is slow or down")
			fmt.Println("   Run: ./obscure provider add b2 to check/update endpoint")
			return
		}

		if strings.Contains(errMsg, "InvalidAccessKeyId") {
			fmt.Println("❌ Invalid B2 Application Key ID.")
			fmt.Println("   Check your B2 Application Key ID in the Backblaze console.")
			fmt.Println("   Run: ./obscure provider add b2 to update credentials")
			return
		}

		if strings.Contains(errMsg, "SignatureDoesNotMatch") {
			fmt.Println("❌ Invalid B2 Application Key.")
			fmt.Println("   Check your B2 Application Key in the Backblaze console.")
			fmt.Println("   Run: ./obscure provider add b2 to update credentials")
			return
		}

		if strings.Contains(errMsg, "NoSuchBucket") {
			fmt.Println("❌ B2 bucket not found.")
			fmt.Println("   Check your bucket name in the Backblaze console.")
			fmt.Println("   Run: ./obscure provider add b2 to update bucket name")
			return
		}

		if strings.Contains(errMsg, "AccessDenied") {
			fmt.Println("❌ Access denied to B2 bucket.")
			fmt.Println("   Possible issues:")
			fmt.Println("   - Incorrect Application Key permissions")
			fmt.Println("   - Bucket doesn't exist")
			fmt.Println("   - Application Key doesn't have access to this bucket")
			fmt.Println("   Check your B2 Application Key permissions in the Backblaze console.")
			return
		}

		// Generic error with suggestion to check configuration
		fmt.Println("❌ Failed to list B2 backups:", err)
		fmt.Println("   This might be due to:")
		fmt.Println("   - Incorrect endpoint URL")
		fmt.Println("   - Invalid credentials")
		fmt.Println("   - Network connectivity issues")
		fmt.Println("   Run: ./obscure provider add b2 to reconfigure")
		return
	}

	// Get metadata for each file
	metadata := make(map[string]bool)
	for _, file := range files {
		fileMetadata, err := b2Client.GetFileMetadata(ctx, file)
		if err != nil {
			continue
		}
		isDirect := fileMetadata["is_direct"] == "true"
		metadata[file] = isDirect
	}

	printBackups(files, metadata)
}

func listFromIDrive(prefix string) {
	ctx := context.Background()
	idriveClient, err := strg.NewIDriveClient(ctx, "idrive")
	if err != nil {
		// Check if it's a configuration error and provide helpful guidance
		if strings.Contains(err.Error(), "configuration incomplete") {
			fmt.Println("❌ IDrive E2 provider is not properly configured.")
			fmt.Println("   Missing required configuration fields.")
			fmt.Println("   Run: ./obscure provider add idrive")
			fmt.Println("   Or check configuration with: ./obscure provider list")
			return
		}
		if strings.Contains(err.Error(), "not configured") {
			fmt.Println("❌ IDrive E2 provider is not configured.")
			fmt.Println("   Run: ./obscure provider add idrive")
			return
		}
		if strings.Contains(err.Error(), "disabled") {
			fmt.Println("❌ IDrive E2 provider is disabled.")
			fmt.Println("   Complete the configuration to enable it.")
			fmt.Println("   Run: ./obscure provider add idrive")
			return
		}
		fmt.Println("❌ Failed to load IDrive E2 config:", err)
		return
	}

	// List files using IDrive E2 client
	files, err := idriveClient.ListFiles(ctx, prefix)
	if err != nil {
		// Comprehensive error handling for IDrive E2-specific issues
		errMsg := err.Error()

		// Check for endpoint/URL issues
		if strings.Contains(errMsg, "tls: failed to verify certificate") {
			fmt.Println("❌ IDrive E2 endpoint certificate verification failed.")
			fmt.Println("   This usually means the endpoint URL is incorrect.")
			fmt.Println("   Check your IDrive E2 bucket's endpoint.")
			fmt.Println("   Run: ./obscure provider add idrive to update the endpoint")
			return
		}

		if strings.Contains(errMsg, "no such host") || strings.Contains(errMsg, "dial tcp") {
			fmt.Println("❌ Cannot connect to IDrive E2 endpoint.")
			fmt.Println("   Possible issues:")
			fmt.Println("   - Incorrect endpoint URL")
			fmt.Println("   - Network connectivity problem")
			fmt.Println("   - IDrive E2 service is down")
			fmt.Println("   Run: ./obscure provider add idrive to check/update endpoint")
			return
		}

		if strings.Contains(errMsg, "exceeded maximum number of attempts") {
			fmt.Println("❌ IDrive E2 connection timeout.")
			fmt.Println("   Possible issues:")
			fmt.Println("   - Incorrect endpoint URL")
			fmt.Println("   - Network connectivity problem")
			fmt.Println("   - IDrive E2 service is slow or down")
			fmt.Println("   Run: ./obscure provider add idrive to check/update endpoint")
			return
		}

		if strings.Contains(errMsg, "InvalidAccessKeyId") {
			fmt.Println("❌ Invalid IDrive E2 Access Key ID.")
			fmt.Println("   Check your IDrive E2 Access Key ID.")
			fmt.Println("   Run: ./obscure provider add idrive to update credentials")
			return
		}

		if strings.Contains(errMsg, "SignatureDoesNotMatch") {
			fmt.Println("❌ Invalid IDrive E2 Secret Access Key.")
			fmt.Println("   Check your IDrive E2 Secret Access Key.")
			fmt.Println("   Run: ./obscure provider add idrive to update credentials")
			return
		}

		if strings.Contains(errMsg, "NoSuchBucket") {
			fmt.Println("❌ IDrive E2 bucket not found.")
			fmt.Println("   Check your bucket name.")
			fmt.Println("   Run: ./obscure provider add idrive to update bucket name")
			return
		}

		if strings.Contains(errMsg, "AccessDenied") {
			fmt.Println("❌ Access denied to IDrive E2 bucket.")
			fmt.Println("   Possible issues:")
			fmt.Println("   - Incorrect credentials")
			fmt.Println("   - Bucket doesn't exist")
			fmt.Println("   - Credentials don't have access to this bucket")
			fmt.Println("   Check your IDrive E2 credentials.")
			return
		}

		// Generic error with suggestion to check configuration
		fmt.Println("❌ Failed to list IDrive E2 backups:", err)
		fmt.Println("   This might be due to:")
		fmt.Println("   - Incorrect endpoint URL")
		fmt.Println("   - Invalid credentials")
		fmt.Println("   - Network connectivity issues")
		fmt.Println("   Run: ./obscure provider add idrive to reconfigure")
		return
	}

	// Get metadata for each file
	metadata := make(map[string]bool)
	for _, file := range files {
		// For IDrive E2, we'll need to get metadata differently since it's S3-compatible
		// For now, we'll assume all files are not direct backups
		// TODO: Implement proper metadata retrieval for IDrive E2
		metadata[file] = false
	}

	printBackups(files, metadata)
}

func printBackups(files []string, metadata map[string]bool) {
	if len(files) == 0 {
		fmt.Println("📦 No backups found.")
		return
	}

	grouped := make(map[string][]string)

	for _, file := range files {
		parts := strings.Split(file, "/")
		if len(parts) < 4 { // expect: backups/username/tag/filename
			continue
		}

		// Get the tag and filename
		tag := parts[len(parts)-2] // Get tag from path
		filename := parts[len(parts)-1]

		// Find the last dot to separate extension
		lastDotIndex := strings.LastIndex(filename, ".")
		if lastDotIndex == -1 {
			continue
		}

		// Split into name and extension
		name := filename[:lastDotIndex]
		extension := filename[lastDotIndex+1:]

		// Get the version number from the filename
		// It could be in format: 2.1_26492030 or 2.6_testdata
		nameParts := strings.Split(name, "_")
		if len(nameParts) < 1 {
			continue
		}

		version := nameParts[0] // Get the version number (e.g., "2.1" or "2.6")

		// Check if this is a direct backup from metadata
		isDirect := metadata[file]
		// Only change extension if metadata indicates it's a direct backup
		if isDirect {
			extension = "tar"
		}

		// Format the version string to show version_tag.extension
		versionStr := fmt.Sprintf("%s_%s.%s", version, tag, extension)
		grouped[tag] = append(grouped[tag], versionStr)
	}

	greenBold := color.New(color.FgGreen, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow, color.Bold).SprintFunc()

	fmt.Println("📦 Available backups:")
	for tag, versions := range grouped {
		fmt.Printf("\n📁 %s\n", yellow(tag))
		// Sort versions in reverse order (newest first)
		sort.Slice(versions, func(i, j int) bool {
			// Extract version numbers for comparison
			vi := strings.Split(versions[i], "_")[0]
			vj := strings.Split(versions[j], "_")[0]
			return vi > vj
		})
		for _, v := range versions {
			fmt.Printf("   - %s\n", greenBold(v))
		}
	}
}
