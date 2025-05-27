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
)

var rmCmd = &cobra.Command{
	Use:   "rm <filename>",
	Short: "Delete a specific backup file",
	Args:  cobra.ExactArgs(1),
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
		filename := args[0]

		// 🔐 Check if filename is missing a tag (no `/` present)
		if !containsSlash(filename) {
			fmt.Println("⚠️  Please specify the full path with tag, e.g., unit/1.0_backup.obscure")
			return
		}

		// 🛑 Ask for confirmation
		fmt.Printf("❓ Are you sure you want to delete %s? (Y/N): ", filename)
		var input string
		fmt.Scanln(&input)
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			fmt.Println("❎ Cancelled deletion.")
			return
		}

		switch providerKey {
		case "gcs":
			key := fmt.Sprintf("backups/%s/%s", username, filename)
			deleteFromGCS(key)
		case "s3":
			key := fmt.Sprintf("backups/%s/%s", username, filename)
			deleteFromS3(key)
		default:
			fmt.Println("❌ Unknown provider:", providerKey)
		}
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}

func deleteFromGCS(key string) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		fmt.Println("❌ GCS client error:", err)
		return
	}
	defer client.Close()

	err = client.Bucket("obscure-open").Object(key).Delete(ctx)
	if err != nil {
		fmt.Println("❌ Failed to delete:", err)
		return
	}
	fmt.Println("🗑️  Deleted:", key)
}

func deleteFromS3(key string) {
	ctx := context.Background()
	cfg, err := configAws()
	if err != nil {
		fmt.Println("❌ AWS config error:", err)
		return
	}
	client := s3sdk.NewFromConfig(cfg)

	_, err = client.DeleteObject(ctx, &s3sdk.DeleteObjectInput{
		Bucket: aws.String("obscure-open"),
		Key:    aws.String(key),
	})
	if err != nil {
		fmt.Println("❌ Failed to delete:", err)
		return
	}
	fmt.Println("🗑️  Deleted:", key)
}

func containsSlash(s string) bool {
	return strings.Contains(s, "/")
}
