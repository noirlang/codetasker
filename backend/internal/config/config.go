// Package config provides application configuration loaded from environment variables.
// It performs validation at startup to ensure all required fields are present,
// preventing silent misconfigurations in production deployments.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config holds all application configuration values loaded from the environment.
// Sensitive fields (e.g. JWTSecret, TokenEncryptKey) must never be logged or
// returned in API responses.
type Config struct {
	// Port is the TCP port the HTTP server listens on (e.g. "8080").
	Port string

	// MongoURI is the full MongoDB connection string.
	MongoURI string

	// DBName is the MongoDB database name to use.
	DBName string

	// GithubClientID is the OAuth App client ID registered on GitHub.
	GithubClientID string

	// GithubClientSecret is the OAuth App client secret. Treat as a password.
	GithubClientSecret string

	// GithubRedirectURL is the callback URL registered in the GitHub OAuth App settings.
	GithubRedirectURL string

	// JWTSecret is the HMAC key used to sign and verify JWT tokens.
	// Must be kept secret and have sufficient entropy.
	JWTSecret string

	// WebhookSecret is the shared secret used to verify GitHub webhook payloads
	// via HMAC-SHA256 signature in the X-Hub-Signature-256 header.
	WebhookSecret string

	// TokenEncryptKey is a 32-byte key used for AES-256-GCM encryption of
	// GitHub access tokens stored in the database. Must be exactly 32 bytes.
	TokenEncryptKey string

	// FrontendURL is the origin of the frontend application, used for CORS
	// and OAuth redirect targets.
	FrontendURL string

	// WebhookProxyURL is an optional public proxy URL (e.g. smee.io or ngrok)
	// used for webhook registration during local development.
	WebhookProxyURL string

	// SMTPHost is the hostname of the SMTP server used for sending emails.
	SMTPHost string

	// SMTPPort is the port of the SMTP server (typically "587" for STARTTLS).
	SMTPPort string

	// SMTPUsername is the SMTP authentication username (usually the sender's email address).
	SMTPUsername string

	// SMTPPassword is the SMTP authentication password or app-specific password.
	SMTPPassword string

	// SMTPFrom is the "From" header value for outgoing emails,
	// e.g. "CodeTasker <noreply@codetasker.dev>".
	SMTPFrom string

	// SMTPEnabled controls whether email notifications are sent.
	// Set SMTP_ENABLED=true in the environment to activate email delivery.
	SMTPEnabled bool
}

// Load reads configuration from environment variables, validates that all required
// fields are present, and returns a populated *Config or a descriptive error listing
// every missing field so operators can fix all problems in one pass.
func Load() (*Config, error) {
	cfg := &Config{
		Port:               os.Getenv("PORT"),
		MongoURI:           os.Getenv("MONGO_URI"),
		DBName:             os.Getenv("DB_NAME"),
		GithubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GithubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		GithubRedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		WebhookSecret:      os.Getenv("WEBHOOK_SECRET"),
		TokenEncryptKey:    os.Getenv("TOKEN_ENCRYPT_KEY"),
		FrontendURL:        os.Getenv("FRONTEND_URL"),
		WebhookProxyURL:    os.Getenv("WEBHOOK_PROXY_URL"),
		// SMTP fields are optional — absent values simply disable email delivery.
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     os.Getenv("SMTP_PORT"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:     os.Getenv("SMTP_FROM"),
		SMTPEnabled:  os.Getenv("SMTP_ENABLED") == "true",
	}

	// Apply sensible default for Port when not specified.
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// Collect all missing required fields so the operator can fix everything at once.
	var missing []string
	required := map[string]string{
		"MONGO_URI":             cfg.MongoURI,
		"DB_NAME":               cfg.DBName,
		"GITHUB_CLIENT_ID":      cfg.GithubClientID,
		"GITHUB_CLIENT_SECRET":  cfg.GithubClientSecret,
		"GITHUB_REDIRECT_URL":   cfg.GithubRedirectURL,
		"JWT_SECRET":            cfg.JWTSecret,
		"WEBHOOK_SECRET":        cfg.WebhookSecret,
		"TOKEN_ENCRYPT_KEY":     cfg.TokenEncryptKey,
		"FRONTEND_URL":          cfg.FrontendURL,
	}

	for key, val := range required {
		if strings.TrimSpace(val) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return nil, errors.New(fmt.Sprintf(
			"missing required environment variables: %s",
			strings.Join(missing, ", "),
		))
	}

	// Validate TOKEN_ENCRYPT_KEY is exactly 32 bytes for AES-256.
	if len(cfg.TokenEncryptKey) != 32 {
		return nil, fmt.Errorf(
			"TOKEN_ENCRYPT_KEY must be exactly 32 bytes for AES-256, got %d bytes",
			len(cfg.TokenEncryptKey),
		)
	}

	return cfg, nil
}
