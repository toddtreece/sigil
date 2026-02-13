package object

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/thanos-io/objstore"
	"github.com/thanos-io/objstore/providers/azure"
	"github.com/thanos-io/objstore/providers/gcs"
	"github.com/thanos-io/objstore/providers/s3"
)

const startupProbeObject = "__sigil_startup_probe__"

type ProviderConfig struct {
	Backend string
	Bucket  string
	S3      S3ProviderConfig
	GCS     GCSProviderConfig
	Azure   AzureProviderConfig
}

type S3ProviderConfig struct {
	Endpoint      string
	Region        string
	AccessKey     string
	SecretKey     string
	Insecure      bool
	UseAWSSDKAuth bool
}

type GCSProviderConfig struct {
	Bucket         string
	ServiceAccount string
	UseGRPC        bool
}

type AzureProviderConfig struct {
	ContainerName           string
	StorageAccountName      string
	StorageAccountKey       string
	StorageConnectionString string
	Endpoint                string
	CreateContainer         bool
}

func NewStoreWithProviderConfig(ctx context.Context, cfg ProviderConfig) (*Store, error) {
	backend := strings.ToLower(strings.TrimSpace(cfg.Backend))
	if backend == "" {
		backend = "s3"
	}

	switch backend {
	case "s3":
		return newS3Store(ctx, cfg)
	case "gcs":
		return newGCSStore(ctx, cfg)
	case "azure":
		return newAzureStore(ctx, cfg)
	default:
		return nil, fmt.Errorf("unsupported object store backend %q", cfg.Backend)
	}
}

// NewStoreWithS3 is preserved for backwards compatibility and local MinIO startup.
func NewStoreWithS3(ctx context.Context, endpoint, bucketName, accessKey, secretKey string, insecure bool) (*Store, error) {
	return NewStoreWithProviderConfig(ctx, ProviderConfig{
		Backend: "s3",
		Bucket:  bucketName,
		S3: S3ProviderConfig{
			Endpoint:  endpoint,
			AccessKey: accessKey,
			SecretKey: secretKey,
			Insecure:  insecure,
		},
	})
}

func newS3Store(ctx context.Context, cfg ProviderConfig) (*Store, error) {
	bucketName := strings.TrimSpace(cfg.Bucket)
	if bucketName == "" {
		return nil, errors.New("object bucket is required")
	}

	normalizedEndpoint, endpointInsecure, err := normalizeS3Endpoint(cfg.S3.Endpoint)
	if err != nil {
		return nil, err
	}
	insecure := cfg.S3.Insecure || endpointInsecure

	accessKey := strings.TrimSpace(cfg.S3.AccessKey)
	secretKey := strings.TrimSpace(cfg.S3.SecretKey)
	if accessKey != "" || secretKey != "" {
		if accessKey == "" || secretKey == "" {
			return nil, errors.New("both object access key and secret key must be provided together")
		}
		if err := ensureBucketExists(ctx, normalizedEndpoint, bucketName, accessKey, secretKey, insecure); err != nil {
			return nil, err
		}
	}

	bucket, err := s3.NewBucketWithConfig(log.NewNopLogger(), s3.Config{
		Bucket:           bucketName,
		Endpoint:         normalizedEndpoint,
		Region:           strings.TrimSpace(cfg.S3.Region),
		AWSSDKAuth:       cfg.S3.UseAWSSDKAuth,
		AccessKey:        accessKey,
		SecretKey:        secretKey,
		Insecure:         insecure,
		BucketLookupType: s3.PathLookup,
	}, "sigil", nil)
	if err != nil {
		return nil, fmt.Errorf("create s3 object bucket client: %w", err)
	}
	if err := validateBucketConnectivity(ctx, bucket); err != nil {
		return nil, err
	}

	store := NewStore(cfg.S3.Endpoint, bucketName)
	store.SetBucket(bucket)
	return store, nil
}

func newGCSStore(ctx context.Context, cfg ProviderConfig) (*Store, error) {
	bucketName := strings.TrimSpace(cfg.GCS.Bucket)
	if bucketName == "" {
		bucketName = strings.TrimSpace(cfg.Bucket)
	}
	if bucketName == "" {
		return nil, errors.New("gcs bucket is required")
	}

	bucket, err := gcs.NewBucketWithConfig(ctx, log.NewNopLogger(), gcs.Config{
		Bucket:         bucketName,
		ServiceAccount: strings.TrimSpace(cfg.GCS.ServiceAccount),
		UseGRPC:        cfg.GCS.UseGRPC,
	}, "sigil", nil)
	if err != nil {
		return nil, fmt.Errorf("create gcs object bucket client: %w", err)
	}
	if err := validateBucketConnectivity(ctx, bucket); err != nil {
		return nil, err
	}

	store := NewStore("gcs://"+bucketName, bucketName)
	store.SetBucket(bucket)
	return store, nil
}

func newAzureStore(ctx context.Context, cfg ProviderConfig) (*Store, error) {
	containerName := strings.TrimSpace(cfg.Azure.ContainerName)
	if containerName == "" {
		containerName = strings.TrimSpace(cfg.Bucket)
	}
	if containerName == "" {
		return nil, errors.New("azure container is required")
	}
	if strings.TrimSpace(cfg.Azure.StorageAccountName) == "" {
		return nil, errors.New("azure storage account is required")
	}

	azureCfg := azure.Config{
		StorageAccountName:      strings.TrimSpace(cfg.Azure.StorageAccountName),
		StorageAccountKey:       strings.TrimSpace(cfg.Azure.StorageAccountKey),
		StorageConnectionString: strings.TrimSpace(cfg.Azure.StorageConnectionString),
		ContainerName:           containerName,
		StorageCreateContainer:  cfg.Azure.CreateContainer,
	}
	if endpoint := strings.TrimSpace(cfg.Azure.Endpoint); endpoint != "" {
		azureCfg.Endpoint = endpoint
	}

	bucket, err := azure.NewBucketWithConfig(log.NewNopLogger(), azureCfg, "sigil", nil)
	if err != nil {
		return nil, fmt.Errorf("create azure object bucket client: %w", err)
	}
	if err := validateBucketConnectivity(ctx, bucket); err != nil {
		return nil, err
	}

	store := NewStore("azure://"+containerName, containerName)
	store.SetBucket(bucket)
	return store, nil
}

func ensureBucketExists(ctx context.Context, endpoint, bucketName, accessKey, secretKey string, insecure bool) error {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: !insecure,
	})
	if err != nil {
		return fmt.Errorf("create minio client: %w", err)
	}

	deadline := time.Now().Add(30 * time.Second)
	for {
		exists, err := client.BucketExists(ctx, bucketName)
		if err == nil {
			if exists {
				return nil
			}
			if createErr := client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); createErr == nil {
				return nil
			}
		}

		if time.Now().After(deadline) {
			if err != nil {
				return fmt.Errorf("check object bucket existence: %w", err)
			}
			return fmt.Errorf("create object bucket %q: timeout waiting for readiness", bucketName)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func validateBucketConnectivity(ctx context.Context, bucket objstore.Bucket) error {
	_, err := bucket.Exists(ctx, startupProbeObject)
	if err != nil {
		return fmt.Errorf("validate object bucket connectivity: %w", err)
	}
	return nil
}

func normalizeS3Endpoint(endpoint string) (string, bool, error) {
	cleaned := strings.TrimSpace(endpoint)
	if cleaned == "" {
		return "", false, errors.New("object endpoint is required")
	}
	if !strings.Contains(cleaned, "://") {
		return cleaned, false, nil
	}

	parsed, err := url.Parse(cleaned)
	if err != nil {
		return "", false, fmt.Errorf("parse object endpoint: %w", err)
	}
	if parsed.Host == "" {
		return "", false, errors.New("object endpoint host is required")
	}
	switch parsed.Scheme {
	case "http":
		return parsed.Host, true, nil
	case "https":
		return parsed.Host, false, nil
	default:
		return "", false, fmt.Errorf("unsupported object endpoint scheme %q", parsed.Scheme)
	}
}
