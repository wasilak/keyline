//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/wasilak/keyline/internal/auth"
	"github.com/wasilak/keyline/internal/config"
)

// setupLDAPContainer creates an OpenLDAP container for testing
func setupLDAPContainer(ctx context.Context, t *testing.T) (testcontainers.Container, string, int) {
	// Use osixia/openldap image with sample data
	req := testcontainers.ContainerRequest{
		Image:        "osixia/openldap:1.5.0",
		ExposedPorts: []string{"389/tcp", "636/tcp"},
		Env: map[string]string{
			"LDAP_ORGANISATION":   "Test Organization",
			"LDAP_DOMAIN":         "example.com",
			"LDAP_ADMIN_PASSWORD": "admin",
			"LDAP_BASE_DN":        "dc=example,dc=com",
		},
		WaitingFor: wait.ForLog("slapd starting").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start LDAP container: %v", err)
	}

	// Get the mapped port for LDAP (389)
	mappedPort, err := container.MappedPort(ctx, "389")
	if err != nil {
		t.Fatalf("Failed to get mapped port: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	ldapURL := fmt.Sprintf("ldap://%s:%s", host, mappedPort.Port())

	t.Logf("LDAP container started at %s", ldapURL)

	// Wait a bit more for LDAP to be fully ready
	time.Sleep(2 * time.Second)

	return container, ldapURL, mappedPort.Int()
}

// TestLDAP_Authentication_Success tests successful LDAP authentication
func TestLDAP_Authentication_Success(t *testing.T) {
	ctx := context.Background()

	// Start LDAP container
	container, ldapURL, _ := setupLDAPContainer(ctx, t)
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Configure LDAP provider
	cfg := &config.LDAPConfig{
		Enabled:              true,
		URL:                  ldapURL,
		BindDN:               "cn=admin,dc=example,dc=com",
		BindPassword:         "${LDAP_ADMIN_PASSWORD}",
		SearchBase:           "dc=example,dc=com",
		SearchFilter:         "(cn={username})",
		UsernameAttribute:    "cn",
		EmailAttribute:       "mail",
		DisplayNameAttribute: "displayName",
		GroupNameAttribute:   "cn",
		ConnectionTimeout:    10 * time.Second,
	}

	// Set environment variable for bind password
	t.Setenv("LDAP_ADMIN_PASSWORD", "admin")

	// Create LDAP provider
	provider, err := auth.NewLDAPProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create LDAP provider: %v", err)
	}

	// The default osixia/openldap image comes with pre-configured users:
	// - cn=admin,dc=example,dc=com (admin user)
	// We need to add a test user or use the admin for testing

	// Test authentication with admin user
	authReq := &auth.AuthRequest{
		AuthorizationHeader: "Basic " + basicAuth("admin", "admin"),
		OriginalURL:         "/",
	}

	result := provider.Authenticate(ctx, authReq)

	if !result.Authenticated {
		t.Errorf("Expected authentication to succeed, got error: %v", result.Error)
	}
	if result.Username != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", result.Username)
	}
}

// TestLDAP_Authentication_InvalidCredentials tests authentication with wrong password
func TestLDAP_Authentication_InvalidCredentials(t *testing.T) {
	ctx := context.Background()

	container, ldapURL, _ := setupLDAPContainer(ctx, t)
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	cfg := &config.LDAPConfig{
		Enabled:           true,
		URL:               ldapURL,
		BindDN:            "cn=admin,dc=example,dc=com",
		BindPassword:      "${LDAP_ADMIN_PASSWORD}",
		SearchBase:        "dc=example,dc=com",
		SearchFilter:      "(cn={username})",
		ConnectionTimeout: 10 * time.Second,
	}

	t.Setenv("LDAP_ADMIN_PASSWORD", "admin")

	provider, err := auth.NewLDAPProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create LDAP provider: %v", err)
	}

	// Try to authenticate with wrong password
	authReq := &auth.AuthRequest{
		AuthorizationHeader: "Basic " + basicAuth("admin", "wrongpassword"),
		OriginalURL:         "/",
	}

	result := provider.Authenticate(ctx, authReq)

	if result.Authenticated {
		t.Error("Expected authentication to fail with wrong password")
	}
	if result.Error == nil {
		t.Error("Expected error for invalid credentials")
	}
}

// TestLDAP_Authentication_UserNotFound tests authentication for non-existent user
func TestLDAP_Authentication_UserNotFound(t *testing.T) {
	ctx := context.Background()

	container, ldapURL, _ := setupLDAPContainer(ctx, t)
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	cfg := &config.LDAPConfig{
		Enabled:           true,
		URL:               ldapURL,
		BindDN:            "cn=admin,dc=example,dc=com",
		BindPassword:      "${LDAP_ADMIN_PASSWORD}",
		SearchBase:        "dc=example,dc=com",
		SearchFilter:      "(cn={username})",
		ConnectionTimeout: 10 * time.Second,
	}

	t.Setenv("LDAP_ADMIN_PASSWORD", "admin")

	provider, err := auth.NewLDAPProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create LDAP provider: %v", err)
	}

	// Try to authenticate with non-existent user
	authReq := &auth.AuthRequest{
		AuthorizationHeader: "Basic " + basicAuth("nonexistentuser", "password"),
		OriginalURL:         "/",
	}

	result := provider.Authenticate(ctx, authReq)

	if result.Authenticated {
		t.Error("Expected authentication to fail for non-existent user")
	}
}

// TestLDAP_TLS_SkipVerify tests LDAPS connection with TLS verification disabled
func TestLDAP_TLS_SkipVerify(t *testing.T) {
	ctx := context.Background()

	container, _, _ := setupLDAPContainer(ctx, t)
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Get the LDAPS port (636)
	ldapsPort, err := container.MappedPort(ctx, "636")
	if err != nil {
		t.Fatalf("Failed to get LDAPS mapped port: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	ldapsURL := fmt.Sprintf("ldaps://%s:%s", host, ldapsPort.Port())

	cfg := &config.LDAPConfig{
		Enabled:           true,
		URL:               ldapsURL,
		BindDN:            "cn=admin,dc=example,dc=com",
		BindPassword:      "${LDAP_ADMIN_PASSWORD}",
		SearchBase:        "dc=example,dc=com",
		SearchFilter:      "(cn={username})",
		TLSMode:           "ldaps",
		TLSSkipVerify:     true, // Skip TLS verification for testing
		ConnectionTimeout: 10 * time.Second,
	}

	t.Setenv("LDAP_ADMIN_PASSWORD", "admin")

	provider, err := auth.NewLDAPProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create LDAP provider: %v", err)
	}

	authReq := &auth.AuthRequest{
		AuthorizationHeader: "Basic " + basicAuth("admin", "admin"),
		OriginalURL:         "/",
	}

	result := provider.Authenticate(ctx, authReq)

	if !result.Authenticated {
		t.Errorf("Expected LDAPS authentication to succeed, got error: %v", result.Error)
	}
}

// TestLDAP_ConnectionTimeout tests connection timeout handling
func TestLDAP_ConnectionTimeout(t *testing.T) {
	ctx := context.Background()

	// Use a non-routable IP to simulate timeout
	cfg := &config.LDAPConfig{
		Enabled:           true,
		URL:               "ldap://192.0.2.1:389", // TEST-NET-1, should not be routable
		BindDN:            "cn=admin,dc=example,dc=com",
		BindPassword:      "${LDAP_ADMIN_PASSWORD}",
		SearchBase:        "dc=example,dc=com",
		SearchFilter:      "(cn={username})",
		ConnectionTimeout: 2 * time.Second, // Short timeout for test
	}

	t.Setenv("LDAP_ADMIN_PASSWORD", "admin")

	provider, err := auth.NewLDAPProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create LDAP provider: %v", err)
	}

	authReq := &auth.AuthRequest{
		AuthorizationHeader: "Basic " + basicAuth("admin", "admin"),
		OriginalURL:         "/",
	}

	// This should timeout
	start := time.Now()
	result := provider.Authenticate(ctx, authReq)
	elapsed := time.Since(start)

	if result.Authenticated {
		t.Error("Expected authentication to fail with timeout")
	}

	// Should complete within reasonable time (timeout + some buffer)
	if elapsed > 5*time.Second {
		t.Errorf("Authentication took too long: %v", elapsed)
	}

	t.Logf("Connection timeout test completed in %v", elapsed)
}

// basicAuth creates a base64-encoded basic auth string
func basicAuth(username, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
}
