package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/joho/godotenv"
)

type BackupConfig struct {
	FilePath      string
	BucketName    string
	Endpoint      string
	AccessKey     string
	SecretKey     string
	Region        string
	KeepBackups   int
	Prefix        string
}

func main() {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error loading .env file: %v", err)
	}

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := backupFile(config); err != nil {
		log.Fatalf("Backup failed: %v", err)
	}

	log.Println("Backup completed successfully")
}

func loadConfig() (*BackupConfig, error) {
	filePath := os.Getenv("BACKUP_FILE")
	if filePath == "" {
		return nil, fmt.Errorf("BACKUP_FILE environment variable is required")
	}

	bucketName := os.Getenv("S3_BUCKET")
	if bucketName == "" {
		return nil, fmt.Errorf("S3_BUCKET environment variable is required")
	}

	endpoint := os.Getenv("S3_ENDPOINT")
	if endpoint == "" {
		return nil, fmt.Errorf("S3_ENDPOINT environment variable is required")
	}

	accessKey := os.Getenv("S3_ACCESS_KEY")
	if accessKey == "" {
		return nil, fmt.Errorf("S3_ACCESS_KEY environment variable is required")
	}

	secretKey := os.Getenv("S3_SECRET_KEY")
	if secretKey == "" {
		return nil, fmt.Errorf("S3_SECRET_KEY environment variable is required")
	}

	region := os.Getenv("S3_REGION")
	if region == "" {
		region = "us-east-1"
	}

	keepBackups := 7
	if kb := os.Getenv("KEEP_BACKUPS"); kb != "" {
		fmt.Sscanf(kb, "%d", &keepBackups)
	}

	prefix := os.Getenv("BACKUP_PREFIX")
	if prefix == "" {
		prefix = "backups"
	}

	return &BackupConfig{
		FilePath:    filePath,
		BucketName:  bucketName,
		Endpoint:    endpoint,
		AccessKey:   accessKey,
		SecretKey:   secretKey,
		Region:      region,
		KeepBackups: keepBackups,
		Prefix:      prefix,
	}, nil
}

func backupFile(cfg *BackupConfig) error {
	if _, err := os.Stat(cfg.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", cfg.FilePath)
	}

	fileName := filepath.Base(cfg.FilePath)
	timestamp := time.Now().Format("20060102-150405")
	backupKey := fmt.Sprintf("%s/%s-%s", cfg.Prefix, timestamp, fileName)

	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
	})

	fileData, err := os.ReadFile(cfg.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(cfg.BucketName),
		Key:    aws.String(backupKey),
		Body:   bytes.NewReader(fileData),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	log.Printf("Uploaded %s to s3://%s/%s", cfg.FilePath, cfg.BucketName, backupKey)

	if err := cleanupOldBackups(client, cfg); err != nil {
		log.Printf("Warning: failed to cleanup old backups: %v", err)
	}

	return nil
}

func cleanupOldBackups(client *s3.Client, cfg *BackupConfig) error {
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(cfg.BucketName),
		Prefix: aws.String(cfg.Prefix),
	})

	var objects []types.Object
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}
		objects = append(objects, page.Contents...)
	}

	if len(objects) <= cfg.KeepBackups {
		return nil
	}

	sort.Slice(objects, func(i, j int) bool {
		return objects[i].LastModified.Before(*objects[j].LastModified)
	})

	for i := 0; i < len(objects)-cfg.KeepBackups; i++ {
		_, err := client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
			Bucket: aws.String(cfg.BucketName),
			Key:    objects[i].Key,
		})
		if err != nil {
			log.Printf("Failed to delete %s: %v", *objects[i].Key, err)
			continue
		}
		log.Printf("Deleted old backup: %s", *objects[i].Key)
	}

	return nil
}
