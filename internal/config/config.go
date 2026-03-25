package config

import "os"

// Config holds runtime settings loaded from the environment.
type Config struct {
	HTTPAddr    string
	DatabaseURL string

	// SQS (LocalStack or AWS). If SQSEndpoint is empty, the API uses a no-op publisher.
	SQSEndpoint        string
	SQSQueueName       string
	AWSRegion          string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	return Config{
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),

		SQSEndpoint:        os.Getenv("SQS_ENDPOINT"),
		SQSQueueName:       getenv("SQS_QUEUE_NAME", "plateful-orders"),
		AWSRegion:          getenv("AWS_REGION", "us-east-1"),
		AWSAccessKeyID:     getenv("AWS_ACCESS_KEY_ID", "test"),
		AWSSecretAccessKey: getenv("AWS_SECRET_ACCESS_KEY", "test"),
	}
}

// SQSEnabled is true when the API/worker should use SQS (typically LocalStack in dev).
func (c Config) SQSEnabled() bool {
	return c.SQSEndpoint != ""
}

func getenv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
