package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/fatih/color"
	cfg "github.com/shah1011/obscure/internal/config"
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
	client, err := storage.NewClient(ctx)
	if err != nil {
		fmt.Println("❌ Error initializing GCS client:", err)
		return
	}
	defer client.Close()

	bucketName := "obscure-open"
	it := client.Bucket(bucketName).Objects(ctx, &storage.Query{Prefix: prefix})
	files := []string{}

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
	}

	printBackups(files, "gcs")
}

func listFromS3(prefix string) {
	ctx := context.Background()
	awsCfg, err := configAws()
	if err != nil {
		fmt.Println("❌ Failed to load AWS config:", err)
		return
	}

	client := s3sdk.NewFromConfig(awsCfg)

	bucketName := "obscure-open"
	input := &s3sdk.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(prefix),
	}

	paginator := s3sdk.NewListObjectsV2Paginator(client, input)
	files := []string{}

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			fmt.Println("❌ Failed to list S3 backups:", err)
			return
		}
		for _, obj := range page.Contents {
			files = append(files, *obj.Key)
		}
	}

	printBackups(files, "s3")
}

func printBackups(files []string, provider string) {
	if len(files) == 0 {
		fmt.Println("📦 No backups found.")
		return
	}

	grouped := make(map[string][]string)

	for _, file := range files {
		parts := strings.Split(file, "/")

		var tag, versionFile string

		switch provider {
		case "gcs":
			if len(parts) < 3 {
				continue // expect: username/tag/version
			}
			tag = parts[len(parts)-2]
			versionFile = parts[len(parts)-1]

		case "s3":
			if len(parts) < 4 {
				continue // expect: backups/username/tag/version
			}
			tag = parts[len(parts)-2]
			versionFile = parts[len(parts)-1]

		default:
			continue // unknown provider
		}

		grouped[tag] = append(grouped[tag], versionFile)
	}

	greenBold := color.New(color.FgGreen, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow, color.Bold).SprintFunc()

	fmt.Println("📦 Available backups:")
	for tag, versions := range grouped {
		fmt.Printf("\n📁 Tag: %s\n", yellow(tag))
		for _, v := range versions {
			fmt.Printf("   - %s\n", greenBold(v))
		}
		fmt.Println()
	}
}

func configAws() (aws.Config, error) {
	return awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(os.Getenv("AWS_REGION")),
	)
}
