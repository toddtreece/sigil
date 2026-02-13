package object

import (
	"context"
	"strings"
	"testing"
)

func TestNewStoreWithProviderConfigRejectsUnsupportedBackend(t *testing.T) {
	_, err := NewStoreWithProviderConfig(context.Background(), ProviderConfig{
		Backend: "unknown",
	})
	if err == nil {
		t.Fatalf("expected unsupported backend error")
	}
}

func TestNewStoreWithProviderConfigS3ValidatesRequiredFields(t *testing.T) {
	_, err := NewStoreWithProviderConfig(context.Background(), ProviderConfig{
		Backend: "s3",
		Bucket:  "",
		S3: S3ProviderConfig{
			Endpoint: "http://minio:9000",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "object bucket is required") {
		t.Fatalf("expected bucket validation error, got %v", err)
	}

	_, err = NewStoreWithProviderConfig(context.Background(), ProviderConfig{
		Backend: "s3",
		Bucket:  "sigil",
		S3:      S3ProviderConfig{},
	})
	if err == nil || !strings.Contains(err.Error(), "object endpoint is required") {
		t.Fatalf("expected endpoint validation error, got %v", err)
	}
}

func TestNewStoreWithProviderConfigGCSValidatesRequiredFields(t *testing.T) {
	_, err := NewStoreWithProviderConfig(context.Background(), ProviderConfig{
		Backend: "gcs",
	})
	if err == nil || !strings.Contains(err.Error(), "gcs bucket is required") {
		t.Fatalf("expected gcs bucket validation error, got %v", err)
	}
}

func TestNewStoreWithProviderConfigAzureValidatesRequiredFields(t *testing.T) {
	_, err := NewStoreWithProviderConfig(context.Background(), ProviderConfig{
		Backend: "azure",
		Bucket:  "sigil",
		Azure: AzureProviderConfig{
			StorageAccountName: "",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "azure storage account is required") {
		t.Fatalf("expected azure storage account validation error, got %v", err)
	}
}

func TestNormalizeS3Endpoint(t *testing.T) {
	host, insecure, err := normalizeS3Endpoint("http://minio:9000")
	if err != nil {
		t.Fatalf("normalize http endpoint: %v", err)
	}
	if host != "minio:9000" || !insecure {
		t.Fatalf("unexpected normalized endpoint host=%q insecure=%v", host, insecure)
	}

	host, insecure, err = normalizeS3Endpoint("https://example.com")
	if err != nil {
		t.Fatalf("normalize https endpoint: %v", err)
	}
	if host != "example.com" || insecure {
		t.Fatalf("unexpected normalized https endpoint host=%q insecure=%v", host, insecure)
	}
}
