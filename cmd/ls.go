package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
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
		default:
			fmt.Println("❌ Unknown provider:", providerKey)
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
			fmt.Println("❌ Failed to list backups:", err)
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
			fmt.Println("❌ Failed to list S3 backups:", err)
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

func configAws() (aws.Config, error) {
	return awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(os.Getenv("AWS_REGION")),
	)
}
