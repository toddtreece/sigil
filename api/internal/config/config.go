package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr            string
	OTLPGRPCAddr        string
	OTLPHTTPAddr        string
	TempoOTLPEndpoint   string
	StorageBackend      string
	MySQLDSN            string
	ObjectStoreEndpoint string
	ObjectStoreBucket   string
	PayloadMaxBytes     int
}

func FromEnv() Config {
	return Config{
		HTTPAddr:            getEnv("SIGIL_HTTP_ADDR", ":8080"),
		OTLPGRPCAddr:        getEnv("SIGIL_OTLP_GRPC_ADDR", ":4317"),
		OTLPHTTPAddr:        getEnv("SIGIL_OTLP_HTTP_ADDR", ":4318"),
		TempoOTLPEndpoint:   getEnv("SIGIL_TEMPO_OTLP_ENDPOINT", "tempo:4317"),
		StorageBackend:      getEnv("SIGIL_STORAGE_BACKEND", "mysql"),
		MySQLDSN:            getEnv("SIGIL_MYSQL_DSN", "sigil:sigil@tcp(mysql:3306)/sigil?parseTime=true"),
		ObjectStoreEndpoint: getEnv("SIGIL_OBJECT_STORE_ENDPOINT", "http://minio:9000"),
		ObjectStoreBucket:   getEnv("SIGIL_OBJECT_STORE_BUCKET", "sigil"),
		PayloadMaxBytes:     getEnvInt("SIGIL_PAYLOAD_MAX_BYTES", 8192),
	}
}

func getEnv(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
