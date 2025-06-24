package cmd

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	s3sdk "github.com/aws/aws-sdk-go-v2/service/s3"
	cfg "github.com/shah1011/obscure/internal/config"
	strg "github.com/shah1011/obscure/internal/storage"
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

		key := fmt.Sprintf("backups/%s/%s", username, filename)
		bucket, err := strg.GetBucketName(providerKey)
		if err != nil {
			fmt.Printf("❌ Failed to get bucket name: %v\n", err)
			return
		}

		switch providerKey {
		case "gcs":
			deleteFromGCS(bucket, key)
		case "s3":
			deleteFromS3(bucket, key)
		case "b2":
			deleteFromB2(key)
		case "idrive":
			deleteFromIDrive(bucket, key)
		case "s3-compatible":
			deleteFromS3Compatible(bucket, key)
		case "storj":
			deleteFromStorj(bucket, key)
		case "filebase-ipfs":
			deleteFromFilebaseIPFS(bucket, key)
		default:
			fmt.Println("❌ Unknown provider:", providerKey)
		}
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}

func deleteFromGCS(bucket, key string) {
	ctx := context.Background()
	client, err := strg.NewGCSClient(ctx, "gcs")
	if err != nil {
		fmt.Println("❌ GCS client error:", err)
		return
	}
	defer client.Close()

	// Check if object exists first
	_, err = client.Bucket(bucket).Object(key).Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			fmt.Printf("❌ File does not exist: %s\n", key)
		} else {
			fmt.Println("❌ Failed to check file existence:", err)
		}
		return
	}

	err = client.Bucket(bucket).Object(key).Delete(ctx)
	if err != nil {
		fmt.Println("❌ Failed to delete:", err)
		return
	}
	fmt.Println("🗑️  Deleted:", key)
}

func deleteFromS3(bucket, key string) {
	ctx := context.Background()
	awsCfg, err := strg.NewAWSClient(ctx, "s3")
	if err != nil {
		fmt.Println("❌ AWS config error:", err)
		return
	}
	client := s3sdk.NewFromConfig(*awsCfg)

	// Check if object exists first
	_, err = client.HeadObject(ctx, &s3sdk.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "StatusCode: 404") {
			fmt.Printf("❌ File does not exist: %s\n", key)
		} else {
			fmt.Println("❌ Failed to check file existence:", err)
		}
		return
	}

	_, err = client.DeleteObject(ctx, &s3sdk.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		fmt.Println("❌ Failed to delete:", err)
		return
	}
	fmt.Println("🗑️  Deleted:", key)
}

func deleteFromB2(key string) {
	ctx := context.Background()
	b2Client, err := strg.NewB2Client(ctx, "b2")
	if err != nil {
		fmt.Println("❌ B2 config error:", err)
		return
	}

	// Check if object exists first
	exists, err := b2Client.FileExists(ctx, key)
	if err != nil {
		fmt.Println("❌ Failed to check file existence:", err)
		return
	}
	if !exists {
		fmt.Printf("❌ File does not exist: %s\n", key)
		return
	}

	err = b2Client.DeleteFile(ctx, key)
	if err != nil {
		fmt.Println("❌ Failed to delete:", err)
		return
	}
	fmt.Println("🗑️  Deleted:", key)
}

func deleteFromIDrive(bucket, key string) {
	ctx := context.Background()
	idriveClient, err := strg.NewIDriveClient(ctx, "idrive")
	if err != nil {
		fmt.Printf("❌ Failed to initialize IDrive E2 client: %v\n", err)
		return
	}

	err = idriveClient.DeleteFile(ctx, key)
	if err != nil {
		fmt.Printf("❌ Failed to delete from IDrive E2: %v\n", err)
		return
	}

	fmt.Println("🗑️  Deleted:", key)
}

func deleteFromS3Compatible(bucket, key string) {
	ctx := context.Background()
	s3CompatibleClient, err := strg.NewS3CompatibleClient(ctx, "s3-compatible")
	if err != nil {
		fmt.Printf("❌ Failed to initialize S3-compatible client: %v\n", err)
		return
	}

	err = s3CompatibleClient.DeleteFile(ctx, key)
	if err != nil {
		fmt.Printf("❌ Failed to delete from S3-compatible: %v\n", err)
		return
	}

	fmt.Println("🗑️  Deleted:", key)
}

func deleteFromStorj(bucket, key string) {
	ctx := context.Background()
	storjClient, err := strg.NewStorjClient(ctx, "storj")
	if err != nil {
		fmt.Printf("❌ Failed to initialize Storj client: %v\n", err)
		return
	}

	// Delete the object
	err = storjClient.DeleteFile(ctx, key)
	if err != nil {
		fmt.Printf("❌ Failed to delete from Storj: %v\n", err)
		return
	}

	fmt.Println("🗑️  Deleted:", key)
}

func deleteFromFilebaseIPFS(bucket, key string) {
	ctx := context.Background()
	s3CompatibleClient, err := strg.NewS3CompatibleClient(ctx, "filebase-ipfs")
	if err != nil {
		fmt.Printf("❌ Failed to initialize Filebase+IPFS client: %v\n", err)
		return
	}

	err = s3CompatibleClient.DeleteFile(ctx, key)
	if err != nil {
		fmt.Printf("❌ Failed to delete from Filebase+IPFS: %v\n", err)
		return
	}

	fmt.Println("🗑️  Deleted:", key)
}

func containsSlash(s string) bool {
	return strings.Contains(s, "/")
}
