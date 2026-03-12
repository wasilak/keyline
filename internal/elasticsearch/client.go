package elasticsearch

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Client defines the interface for Elasticsearch Security API operations
type Client interface {
	// CreateOrUpdateUser creates or updates an ES user
	CreateOrUpdateUser(ctx context.Context, req *UserRequest) error

	// GetUser retrieves user information
	GetUser(ctx context.Context, username string) (*User, error)

	// DeleteUser deletes an ES user
	DeleteUser(ctx context.Context, username string) error

	// ValidateConnection validates admin credentials and connectivity
	ValidateConnection(ctx context.Context) error
}

// UserRequest represents a request to create or update an ES user
type UserRequest struct {
	Username string
	Password string
	Roles    []string
	FullName string
	Email    string
	Metadata map[string]interface{}
}

// User represents an Elasticsearch user
type User struct {
	Username string
	Roles    []string
	FullName string
	Email    string
	Metadata map[string]interface{}
	Enabled  bool
}

// Config holds the Elasticsearch client configuration
type Config struct {
	URL                string
	AdminUser          string
	AdminPassword      string
	Timeout            time.Duration
	InsecureSkipVerify bool
}

// client implements the Client interface
type client struct {
	httpClient *http.Client
	config     Config
	authHeader string
}

// NewClient creates a new Elasticsearch API client
func NewClient(config Config) (Client, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("elasticsearch URL is required")
	}
	if config.AdminUser == "" {
		return nil, fmt.Errorf("elasticsearch admin user is required")
	}
	if config.AdminPassword == "" {
		return nil, fmt.Errorf("elasticsearch admin password is required")
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// Create HTTP client with TLS configuration
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	// Create Basic Auth header
	auth := base64.StdEncoding.EncodeToString(
		[]byte(fmt.Sprintf("%s:%s", config.AdminUser, config.AdminPassword)),
	)
	authHeader := fmt.Sprintf("Basic %s", auth)

	return &client{
		httpClient: httpClient,
		config:     config,
		authHeader: authHeader,
	}, nil
}

// CreateOrUpdateUser creates or updates an ES user with retry logic and tracing
func (c *client) CreateOrUpdateUser(ctx context.Context, req *UserRequest) error {
	ctx, span := otel.Tracer("keyline").Start(ctx, "elasticsearch.create_or_update_user")
	defer span.End()

	span.SetAttributes(
		attribute.String("es.username", req.Username),
		attribute.StringSlice("es.roles", req.Roles),
	)

	// Build request body
	body := map[string]interface{}{
		"password": req.Password,
		"roles":    req.Roles,
		"enabled":  true,
	}

	if req.FullName != "" {
		body["full_name"] = req.FullName
	}
	if req.Email != "" {
		body["email"] = req.Email
	}
	if req.Metadata != nil && len(req.Metadata) > 0 {
		body["metadata"] = req.Metadata
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to marshal request body")
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Retry logic with exponential backoff
	var lastErr error
	backoff := time.Second

	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			slog.WarnContext(ctx, "Retrying ES API call",
				slog.Int("attempt", attempt),
				slog.Duration("backoff", backoff),
				slog.String("username", req.Username),
			)
			time.Sleep(backoff)
			backoff *= 2
		}

		err := c.doCreateOrUpdateUser(ctx, req.Username, bodyBytes)
		if err == nil {
			// Prometheus metrics - will be called from usermgmt package
			slog.InfoContext(ctx, "ES user created/updated successfully",
				slog.String("username", req.Username),
				slog.Any("roles", req.Roles),
				slog.Int("attempt", attempt),
			)
			return nil
		}

		lastErr = err

		// Don't retry on authentication or authorization errors
		if isAuthError(err) {
			span.RecordError(err)
			span.SetStatus(codes.Error, "authentication/authorization failed")
			return err
		}
	}

	span.RecordError(lastErr)
	span.SetStatus(codes.Error, "all retry attempts failed")
	return fmt.Errorf("failed to create/update user after 3 attempts: %w", lastErr)
}

// doCreateOrUpdateUser performs the actual HTTP request
func (c *client) doCreateOrUpdateUser(ctx context.Context, username string, bodyBytes []byte) error {
	url := fmt.Sprintf("%s/_security/user/%s", c.config.URL, username)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for error messages
	respBody, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return nil
	case http.StatusUnauthorized:
		return &AuthError{StatusCode: resp.StatusCode, Message: "invalid admin credentials"}
	case http.StatusForbidden:
		return &AuthError{StatusCode: resp.StatusCode, Message: "insufficient permissions"}
	case http.StatusTooManyRequests:
		return &RateLimitError{StatusCode: resp.StatusCode, Message: "rate limited"}
	default:
		if resp.StatusCode >= 500 {
			return &ServerError{StatusCode: resp.StatusCode, Message: string(respBody)}
		}
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	}
}

// GetUser retrieves user information from Elasticsearch
func (c *client) GetUser(ctx context.Context, username string) (*User, error) {
	ctx, span := otel.Tracer("keyline").Start(ctx, "elasticsearch.get_user")
	defer span.End()

	span.SetAttributes(attribute.String("es.username", username))

	url := fmt.Sprintf("%s/_security/user/%s", c.config.URL, username)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // User doesn't exist
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
		span.RecordError(err)
		return nil, err
	}

	var result map[string]struct {
		Username string                 `json:"username"`
		Roles    []string               `json:"roles"`
		FullName string                 `json:"full_name"`
		Email    string                 `json:"email"`
		Metadata map[string]interface{} `json:"metadata"`
		Enabled  bool                   `json:"enabled"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// ES returns a map with username as key
	for _, userData := range result {
		return &User{
			Username: userData.Username,
			Roles:    userData.Roles,
			FullName: userData.FullName,
			Email:    userData.Email,
			Metadata: userData.Metadata,
			Enabled:  userData.Enabled,
		}, nil
	}

	return nil, fmt.Errorf("unexpected response format")
}

// DeleteUser deletes an ES user
func (c *client) DeleteUser(ctx context.Context, username string) error {
	ctx, span := otel.Tracer("keyline").Start(ctx, "elasticsearch.delete_user")
	defer span.End()

	span.SetAttributes(attribute.String("es.username", username))

	url := fmt.Sprintf("%s/_security/user/%s", c.config.URL, username)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
		slog.InfoContext(ctx, "ES user deleted", slog.String("username", username))
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	err = fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(respBody))
	span.RecordError(err)
	return err
}

// ValidateConnection validates admin credentials and ES connectivity
func (c *client) ValidateConnection(ctx context.Context) error {
	ctx, span := otel.Tracer("keyline").Start(ctx, "elasticsearch.validate_connection")
	defer span.End()

	url := fmt.Sprintf("%s/_security/_authenticate", c.config.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &AuthError{
			StatusCode: resp.StatusCode,
			Message:    "invalid admin credentials or insufficient permissions",
		}
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("validation failed with status %d: %s", resp.StatusCode, string(respBody))
		span.RecordError(err)
		return err
	}

	slog.InfoContext(ctx, "ES connection validated successfully")
	return nil
}

// Error types for better error handling

// AuthError represents authentication or authorization errors
type AuthError struct {
	StatusCode int
	Message    string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("auth error (status %d): %s", e.StatusCode, e.Message)
}

// RateLimitError represents rate limiting errors
type RateLimitError struct {
	StatusCode int
	Message    string
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit error (status %d): %s", e.StatusCode, e.Message)
}

// ServerError represents server-side errors (5xx)
type ServerError struct {
	StatusCode int
	Message    string
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("server error (status %d): %s", e.StatusCode, e.Message)
}

// isAuthError checks if an error is an authentication/authorization error
func isAuthError(err error) bool {
	_, ok := err.(*AuthError)
	return ok
}
