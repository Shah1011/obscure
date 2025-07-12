package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	cron "github.com/robfig/cron/v3"

	cfg "github.com/shah1011/obscure/internal/config"
	strg "github.com/shah1011/obscure/internal/storage"
	"github.com/shah1011/obscure/utils"
	"github.com/spf13/cobra"
)

var (
	schedTime     string
	schedInterval string // can be "daily", "minute", or "custom"
	schedDir      string
	schedTag      string
	schedVersion  string
	schedRetain   int
)

// runScheduledBackup runs a backup non-interactively for the scheduler
func runScheduledBackup(dir, tag, version string, retain int) error {
	const defaultPassword = "scheduler-default-password" // TODO: Secure this!
	isDirect := false

	if _, err := cfg.GetSessionEmail(); err != nil {
		return fmt.Errorf("not logged in: %w", err)
	}
	username, err := cfg.GetSessionUsername()
	if err != nil {
		return fmt.Errorf("failed to get username: %w", err)
	}
	providerKey, err := cfg.GetSessionProvider()
	if err != nil || providerKey == "" {
		providerKey, err = cfg.GetUserDefaultProvider()
		if err != nil || providerKey == "" {
			return fmt.Errorf("no cloud provider configured")
		}
	}
	providers, err := cfg.LoadUserProviders()
	if err != nil {
		return fmt.Errorf("failed to load provider config: %w", err)
	}

	if version == "auto" || version == "" {
		version = time.Now().Format("2006.01.02-15.04.05")
	}

	backupFile, err := CreateBackupFile(dir)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer os.Remove(backupFile.Name())
	defer backupFile.Close()

	var uploadReader io.Reader
	var extension string

	if isDirect {
		uploadReader = backupFile
		extension = "tar"
	} else {
		var encryptedBuf bytes.Buffer
		encWriter, err := utils.EncryptStream(&encryptedBuf, defaultPassword)
		if err != nil {
			return fmt.Errorf("failed to initialize encryption: %w", err)
		}
		compWriter := utils.NewCompressWriter(encWriter)
		defer compWriter.Close()
		if _, err := io.Copy(compWriter, backupFile); err != nil {
			return fmt.Errorf("failed to compress and encrypt: %w", err)
		}
		if err := compWriter.Close(); err != nil {
			return fmt.Errorf("failed to close compression: %w", err)
		}
		if err := encWriter.Close(); err != nil {
			return fmt.Errorf("failed to finalize encryption: %w", err)
		}
		uploadReader = bytes.NewReader(encryptedBuf.Bytes())
		extension = "obscure"
	}

	filename := fmt.Sprintf("%s_%s.%s", version, tag, extension)
	key := fmt.Sprintf("backups/%s/%s/%s", username, tag, filename)

	config, ok := providers.Providers[providerKey]
	if !ok || !config.Enabled {
		return fmt.Errorf("provider %s is not configured or disabled", providerKey)
	}

	awsCfg, err := strg.NewAWSClient(context.Background(), "s3")
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}
	client := s3.NewFromConfig(*awsCfg)
	_, err = client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(config.Bucket),
		Key:    aws.String(key),
		Body:   uploadReader,
		Metadata: map[string]string{
			"username":  username,
			"tag":       tag,
			"version":   version,
			"is_direct": fmt.Sprintf("%v", isDirect),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}

	fmt.Printf("[Scheduler] Backup completed: %s\n", key)
	enforceRetention(username, tag, retain, config)
	return nil
}

// enforceRetention deletes oldest backups if over the retain limit (S3 only)
func enforceRetention(username, tag string, retain int, config *cfg.CloudProviderConfig) error {
	ctx := context.Background()
	awsCfg, err := strg.NewAWSClient(ctx, "s3")
	if err != nil {
		return fmt.Errorf("failed to init AWS client: %w", err)
	}
	client := s3.NewFromConfig(*awsCfg)
	prefix := fmt.Sprintf("backups/%s/%s/", username, tag)
	var backups []string
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(config.Bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list backups: %w", err)
		}
		for _, obj := range page.Contents {
			backups = append(backups, *obj.Key)
		}
	}
	if len(backups) <= retain {
		return nil // nothing to delete
	}
	// Sort backups by key (filename has timestamp/version prefix)
	sort.Strings(backups)
	toDelete := backups[:len(backups)-retain]
	for _, key := range toDelete {
		_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(config.Bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			fmt.Printf("[Scheduler] Failed to delete old backup: %s (%v)\n", key, err)
		} else {
			fmt.Printf("[Scheduler] Deleted old backup: %s\n", key)
		}
	}
	return nil
}

func scheduleBackupJob() {
	c := cron.New()
	var cronExpr string
	switch schedInterval {
	case "minute":
		// --time=N means every N minutes
		cronExpr = fmt.Sprintf("*/%s * * * *", schedTime)
	case "custom":
		// --time is a full cron expression
		cronExpr = schedTime
	default:
		// Default: daily at HH:MM
		hourMin := strings.Split(schedTime, ":")
		if len(hourMin) != 2 {
			fmt.Println("[Scheduler] Invalid time format. Use HH:MM.")
			return
		}
		cronExpr = fmt.Sprintf("%s %s * * *", hourMin[1], hourMin[0])
	}
	fmt.Printf("[Scheduler] Using cron expression: %s\n", cronExpr)
	_, err := c.AddFunc(cronExpr, func() {
		err := runScheduledBackup(schedDir, schedTag, schedVersion, schedRetain)
		if err != nil {
			fmt.Printf("[Scheduler] Backup failed: %v\n", err)
		}
	})
	if err != nil {
		fmt.Printf("[Scheduler] Failed to schedule job: %v\n", err)
		return
	}
	fmt.Println("[Scheduler] Starting scheduler...")
	c.Start()
	select {} // Block forever
}

var schedulerCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "Schedule automated backups at specified intervals.",
	Long:  `Automate backups with a scheduler.\n\n- The scheduler always uses the currently selected provider (set with 'obscure switch-provider') at the time of each backup.\n- To change the provider for future scheduled backups, run 'obscure switch-provider <provider>' before the next backup runs.\n\nExamples:\n  Daily at 17:00: obscure scheduler --time=\"17:00\" --interval=daily ...\n  Every 5 minutes: obscure scheduler --time=\"5\" --interval=minute ...\n  Custom cron: obscure scheduler --time=\"*/10 * * * *\" --interval=custom ...`,
	Run: func(cmd *cobra.Command, args []string) {
		if schedTime == "" || schedInterval == "" || schedDir == "" || schedTag == "" {
			fmt.Println("❌ --time, --interval, --dir, and --tag are required.")
			return
		}
		if schedVersion == "" {
			schedVersion = "auto"
		}
		if schedRetain == 0 {
			schedRetain = 5
		}
		fmt.Printf("[Scheduler] Scheduling backup: time=%s, interval=%s, dir=%s, tag=%s, version=%s, retain=%d\n", schedTime, schedInterval, schedDir, schedTag, schedVersion, schedRetain)
		scheduleBackupJob()
	},
}

func init() {
	rootCmd.AddCommand(schedulerCmd)
	schedulerCmd.Flags().StringVar(&schedTime, "time", "", "Time of day to run the backup (e.g., 17:00)")
	schedulerCmd.Flags().StringVar(&schedInterval, "interval", "", "Interval for backup (e.g., daily, weekly, minute, custom)")
	schedulerCmd.Flags().StringVar(&schedDir, "dir", "", "Directory to back up")
	schedulerCmd.Flags().StringVar(&schedTag, "tag", "", "Tag for the backup")
	schedulerCmd.Flags().StringVar(&schedVersion, "version", "auto", "Backup version (auto-increment if not specified)")
	schedulerCmd.Flags().IntVar(&schedRetain, "retain", 5, "Retention policy (number of backups to keep, default 5)")
}
