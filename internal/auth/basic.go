package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"

	"github.com/yourusername/keyline/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// BasicAuthProvider implements Basic Auth for local users
type BasicAuthProvider struct {
	config *config.LocalUsersConfig
}

// NewBasicAuthProvider creates a new Basic Auth provider
func NewBasicAuthProvider(cfg *config.LocalUsersConfig) (*BasicAuthProvider, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("local users authentication is not enabled")
	}

	if len(cfg.Users) == 0 {
		return nil, fmt.Errorf("no local users configured")
	}

	return &BasicAuthProvider{
		config: cfg,
	}, nil
}

// AuthRequest contains authentication request data
type AuthRequest struct {
	AuthorizationHeader string
	OriginalURL         string
}

// AuthResult contains authentication result
type AuthResult struct {
	Authenticated bool
	Username      string
	ESUser        string
	Error         error
}

// Authenticate validates Basic Auth credentials
func (p *BasicAuthProvider) Authenticate(ctx context.Context, req *AuthRequest) *AuthResult {
	slog.InfoContext(ctx, "Attempting Basic Auth authentication")

	// Extract Authorization header
	if req.AuthorizationHeader == "" {
		slog.WarnContext(ctx, "Missing Authorization header")
		return &AuthResult{
			Authenticated: false,
			Error:         fmt.Errorf("missing Authorization header"),
		}
	}

	// Check if it's Basic auth
	if !strings.HasPrefix(req.AuthorizationHeader, "Basic ") {
		slog.WarnContext(ctx, "Authorization header is not Basic auth")
		return &AuthResult{
			Authenticated: false,
			Error:         fmt.Errorf("not Basic auth"),
		}
	}

	// Extract base64-encoded credentials
	encodedCreds := strings.TrimPrefix(req.AuthorizationHeader, "Basic ")
	if encodedCreds == "" {
		slog.WarnContext(ctx, "Empty Basic auth credentials")
		return &AuthResult{
			Authenticated: false,
			Error:         fmt.Errorf("empty credentials"),
		}
	}

	// Decode base64 credentials
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedCreds)
	if err != nil {
		slog.WarnContext(ctx, "Failed to decode Basic auth credentials",
			slog.String("error", err.Error()),
		)
		return &AuthResult{
			Authenticated: false,
			Error:         fmt.Errorf("invalid base64 encoding"),
		}
	}

	decodedCreds := string(decodedBytes)

	// Extract username and password
	username, password, err := p.extractCredentials(decodedCreds)
	if err != nil {
		slog.WarnContext(ctx, "Failed to extract credentials",
			slog.String("error", err.Error()),
		)
		return &AuthResult{
			Authenticated: false,
			Error:         err,
		}
	}

	// Look up user
	user := p.findUser(username)
	if user == nil {
		slog.WarnContext(ctx, "User not found",
			slog.String("username", username),
		)
		return &AuthResult{
			Authenticated: false,
			Username:      username,
			Error:         fmt.Errorf("user not found"),
		}
	}

	// Validate password using bcrypt
	if err := p.validatePassword(ctx, password, user.PasswordBcrypt); err != nil {
		slog.WarnContext(ctx, "Password validation failed",
			slog.String("username", username),
		)
		return &AuthResult{
			Authenticated: false,
			Username:      username,
			Error:         fmt.Errorf("invalid password"),
		}
	}

	// Authentication successful
	slog.InfoContext(ctx, "Basic Auth authentication successful",
		slog.String("username", username),
		slog.String("es_user", user.ESUser),
	)

	return &AuthResult{
		Authenticated: true,
		Username:      username,
		ESUser:        user.ESUser,
		Error:         nil,
	}
}

// extractCredentials splits decoded credentials on ":" separator
func (p *BasicAuthProvider) extractCredentials(decodedCreds string) (username, password string, err error) {
	// Split on first ":" to handle passwords containing ":"
	parts := strings.SplitN(decodedCreds, ":", 2)

	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid credentials format: missing separator")
	}

	username = parts[0]
	password = parts[1]

	// Validate non-empty
	if username == "" {
		return "", "", fmt.Errorf("empty username")
	}

	if password == "" {
		return "", "", fmt.Errorf("empty password")
	}

	return username, password, nil
}

// findUser searches for a user by username
func (p *BasicAuthProvider) findUser(username string) *config.LocalUser {
	for i := range p.config.Users {
		if p.config.Users[i].Username == username {
			return &p.config.Users[i]
		}
	}
	return nil
}

// validatePassword validates the password using bcrypt timing-safe comparison
func (p *BasicAuthProvider) validatePassword(ctx context.Context, password, passwordBcrypt string) error {
	// Use bcrypt.CompareHashAndPassword for timing-safe comparison
	err := bcrypt.CompareHashAndPassword([]byte(passwordBcrypt), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return fmt.Errorf("password mismatch")
		}
		// Log unexpected errors but don't expose details
		slog.ErrorContext(ctx, "Unexpected bcrypt error",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("password validation error")
	}
	return nil
}
