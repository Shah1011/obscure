package cmd

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	cfg "github.com/shah1011/obscure/internal/config"
	"github.com/spf13/cobra"
	"google.golang.org/api/iterator"
)

var rmdirCmd = &cobra.Command{
	Use:   "rmdir <tag>",
	Short: "Delete all backup files under a specific tag for the current user",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		tag := args[0]
		if tag == "" {
			fmt.Println("⚠️  Please provide a valid tag name, e.g., `unit` or `testdata`")
			return
		}

		providerKey, err := cfg.GetSessionProvider()
		if err != nil || providerKey == "" {
			providerKey, err = cfg.GetUserDefaultProvider()
			if err != nil || providerKey == "" {
				fmt.Println("⚠️  No cloud provider is configured.")
				return
			}
		}

		username, _ := cfg.GetSessionUsername()

		var prefix string
		switch providerKey {
		case "gcs":
			prefix = fmt.Sprintf("backups/%s/%s/", username, tag)
		case "s3":
			prefix = fmt.Sprintf("backups/%s/%s/", username, tag)
		default:
			fmt.Println("❌ Unknown provider:", providerKey)
			return
		}

		// Check if the tag exists (at least one object with prefix)
		exists, err := tagExists(providerKey, prefix)
		if err != nil {
			fmt.Println("❌ Error checking tag existence:", err)
			return
		}
		if !exists {
			fmt.Printf("⚠️ Tag '%s' does not exist for user '%s'\n", tag, username)
			return
		}

		// Confirmation prompt
		fmt.Printf("❓ Are you sure you want to delete ALL backups under tag '%s'? This action is irreversible. (Y/N): ", tag)
		var input string
		fmt.Scanln(&input)
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			fmt.Println("❎ Cancelled deletion.")
			return
		}

		// Proceed with deletion
		switch providerKey {
		case "gcs":
			deleteAllFromGCS(prefix)
		case "s3":
			deleteAllFromS3(prefix)
		}
	},
}

func init() {
	rootCmd.AddCommand(rmdirCmd)
}

// tagExists checks if any objects exist under the given prefix for the provider.
func tagExists(providerKey, prefix string) (bool, error) {
	switch providerKey {
	case "gcs":
		ctx := context.Background()
		client, err := storage.NewClient(ctx)
		if err != nil {
			return false, fmt.Errorf("GCS client error: %w", err)
		}
		defer client.Close()

		it := client.Bucket("obscure-open").Objects(ctx, &storage.Query{
			Prefix: prefix,
		})
		_, err = it.Next()
		if err == iterator.Done {
			return false, nil // no objects found
		}
		if err != nil {
			return false, fmt.Errorf("error during listing: %w", err)
		}
		return true, nil

	case "s3":
		ctx := context.Background()
		awsCfg, err := configAws()
		if err != nil {
			return false, fmt.Errorf("AWS config error: %w", err)
		}
		client := s3sdk.NewFromConfig(awsCfg)

		resp, err := client.ListObjectsV2(ctx, &s3sdk.ListObjectsV2Input{
			Bucket:  aws.String("obscure-open"),
			Prefix:  aws.String(prefix),
			MaxKeys: aws.Int32(1),
		})
		if err != nil {
			return false, fmt.Errorf("error during listing: %w", err)
		}
		return len(resp.Contents) > 0, nil

	default:
		return false, fmt.Errorf("unknown provider: %s", providerKey)
	}
}

func deleteAllFromGCS(prefix string) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		fmt.Println("❌ GCS client error:", err)
		return
	}
	defer client.Close()

	it := client.Bucket("obscure-open").Objects(ctx, &storage.Query{Prefix: prefix})
	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println("❌ Error during listing:", err)
			return
		}
		err = client.Bucket("obscure-open").Object(obj.Name).Delete(ctx)
		if err != nil {
			fmt.Println("❌ Failed to delete:", obj.Name)
			continue
		}
		fmt.Println("🗑️ Deleted:", obj.Name)
	}
}

func deleteAllFromS3(prefix string) {
	ctx := context.Background()
	cfg, err := configAws()
	if err != nil {
		fmt.Println("❌ AWS config error:", err)
		return
	}
	client := s3sdk.NewFromConfig(cfg)

	paginator := s3sdk.NewListObjectsV2Paginator(client, &s3sdk.ListObjectsV2Input{
		Bucket: aws.String("obscure-open"),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			fmt.Println("❌ Error during listing:", err)
			return
		}
		for _, obj := range page.Contents {
			_, err := client.DeleteObject(ctx, &s3sdk.DeleteObjectInput{
				Bucket: aws.String("obscure-open"),
				Key:    obj.Key,
			})
			if err != nil {
				fmt.Println("❌ Failed to delete:", *obj.Key)
				continue
			}
			fmt.Println("🗑️ Deleted:", *obj.Key)
		}
	}
}
