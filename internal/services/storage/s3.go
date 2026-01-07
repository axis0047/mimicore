package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketName      string
}

type Client struct {
	minioClient *minio.Client
	bucket      string
}

var GlobalStorage *Client

// Init initializes the S3 client globally
func Init(cfg S3Config) error {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return err
	}

	// Ensure bucket exists
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.BucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.BucketName, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}

	GlobalStorage = &Client{
		minioClient: client,
		bucket:      cfg.BucketName,
	}
	log.Println("✅ S3 Storage Initialized")
	return nil
}

// Get attempts to fetch the compiled WASM from S3
func (s *Client) Get(hash string) ([]byte, bool) {
	ctx := context.Background()
	objectName := fmt.Sprintf("wasm-cache/%s.wasm", hash)

	obj, err := s.minioClient.GetObject(ctx, s.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, false
	}
	defer obj.Close()

	// Check if object exists (GetObject is lazy)
	_, err = obj.Stat()
	if err != nil {
		return nil, false
	}

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, false
	}
	return data, true
}

// Put uploads the compiled WASM to S3
func (s *Client) Put(hash string, data []byte) error {
	ctx := context.Background()
	objectName := fmt.Sprintf("wasm-cache/%s.wasm", hash)
	reader := bytes.NewReader(data)

	_, err := s.minioClient.PutObject(ctx, s.bucket, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/wasm",
	})
	return err
}

// List returns all object names in the cache folder
func (s *Client) List() ([]string, error) {
	ctx := context.Background()
	var results []string

	opts := minio.ListObjectsOptions{
		Prefix:    "wasm-cache/",
		Recursive: true,
	}

	for object := range s.minioClient.ListObjects(ctx, s.bucket, opts) {
		if object.Err != nil {
			return nil, object.Err
		}
		results = append(results, object.Key)
	}
	return results, nil
}

// Delete removes objects
func (s *Client) Delete(keys []string) error {
	ctx := context.Background()
	objectsCh := make(chan minio.ObjectInfo)

	// Send objects to be deleted
	go func() {
		defer close(objectsCh)
		for _, key := range keys {
			objectsCh <- minio.ObjectInfo{Key: key}
		}
	}()

	opts := minio.RemoveObjectsOptions{GovernanceBypass: true}
	for err := range s.minioClient.RemoveObjects(ctx, s.bucket, objectsCh, opts) {
		log.Printf("Failed to delete %s: %v", err.ObjectName, err.Err)
	}
	return nil
}
