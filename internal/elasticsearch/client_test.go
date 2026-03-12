package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockESServer creates a test HTTP server that mocks Elasticsearch responses
func mockESServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

// TestNewClient tests client creation
func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				URL:           "https://localhost:9200",
				AdminUser:     "admin",
				AdminPassword: "password",
				Timeout:       30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing URL",
			config: Config{
				AdminUser:     "admin",
				AdminPassword: "password",
			},
			wantErr: true,
			errMsg:  "elasticsearch URL is required",
		},
		{
			name: "missing admin user",
			config: Config{
				URL:           "https://localhost:9200",
				AdminPassword: "password",
			},
			wantErr: true,
			errMsg:  "elasticsearch admin user is required",
		},
		{
			name: "missing admin password",
			config: Config{
				URL:       "https://localhost:9200",
				AdminUser: "admin",
			},
			wantErr: true,
			errMsg:  "elasticsearch admin password is required",
		},
		{
			name: "default timeout",
			config: Config{
				URL:           "https://localhost:9200",
				AdminUser:     "admin",
				AdminPassword: "password",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewClient() expected error, got nil")
				} else if err.Error() != tt.errMsg {
					t.Errorf("NewClient() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("NewClient() unexpected error: %v", err)
				}
				if client == nil {
					t.Errorf("NewClient() returned nil client")
				}
			}
		})
	}
}

// TestCreateOrUpdateUser_Success tests successful user creation
func TestCreateOrUpdateUser_Success(t *testing.T) {
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and path
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT request, got %s", r.Method)
		}
		if r.URL.Path != "/_security/user/testuser" {
			t.Errorf("Expected path /_security/user/testuser, got %s", r.URL.Path)
		}

		// Verify Authorization header
		if r.Header.Get("Authorization") == "" {
			t.Error("Missing Authorization header")
		}

		// Verify Content-Type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Parse and verify request body
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if body["password"] == "" {
			t.Error("Missing password in request body")
		}
		if body["enabled"] != true {
			t.Error("Expected enabled=true in request body")
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"created": true}`))
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	req := &UserRequest{
		Username: "testuser",
		Password: "testpassword",
		Roles:    []string{"admin", "kibana_user"},
		FullName: "Test User",
		Email:    "test@example.com",
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	err = client.CreateOrUpdateUser(ctx, req)
	if err != nil {
		t.Errorf("CreateOrUpdateUser() unexpected error: %v", err)
	}
}

// TestCreateOrUpdateUser_Update tests user update (200 OK)
func TestCreateOrUpdateUser_Update(t *testing.T) {
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"created": false}`))
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	req := &UserRequest{
		Username: "existinguser",
		Password: "newpassword",
		Roles:    []string{"viewer"},
	}

	err = client.CreateOrUpdateUser(ctx, req)
	if err != nil {
		t.Errorf("CreateOrUpdateUser() unexpected error: %v", err)
	}
}

// TestCreateOrUpdateUser_Unauthorized tests invalid admin credentials (401)
func TestCreateOrUpdateUser_Unauthorized(t *testing.T) {
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "wrongpassword",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	req := &UserRequest{
		Username: "testuser",
		Password: "testpassword",
		Roles:    []string{"admin"},
	}

	err = client.CreateOrUpdateUser(ctx, req)
	if err == nil {
		t.Error("CreateOrUpdateUser() expected error, got nil")
	}

	// Verify it's an AuthError
	authErr, ok := err.(*AuthError)
	if !ok {
		t.Errorf("Expected AuthError, got %T", err)
	} else {
		if authErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, authErr.StatusCode)
		}
	}
}

// TestCreateOrUpdateUser_Forbidden tests insufficient permissions (403)
func TestCreateOrUpdateUser_Forbidden(t *testing.T) {
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "forbidden"}`))
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	req := &UserRequest{
		Username: "testuser",
		Password: "testpassword",
		Roles:    []string{"admin"},
	}

	err = client.CreateOrUpdateUser(ctx, req)
	if err == nil {
		t.Error("CreateOrUpdateUser() expected error, got nil")
	}

	// Verify it's an AuthError
	authErr, ok := err.(*AuthError)
	if !ok {
		t.Errorf("Expected AuthError, got %T", err)
	} else {
		if authErr.StatusCode != http.StatusForbidden {
			t.Errorf("Expected status code %d, got %d", http.StatusForbidden, authErr.StatusCode)
		}
	}
}

// TestCreateOrUpdateUser_RateLimited tests rate limiting (429)
func TestCreateOrUpdateUser_RateLimited(t *testing.T) {
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limited"}`))
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	req := &UserRequest{
		Username: "testuser",
		Password: "testpassword",
		Roles:    []string{"admin"},
	}

	err = client.CreateOrUpdateUser(ctx, req)
	if err == nil {
		t.Error("CreateOrUpdateUser() expected error, got nil")
	}

	// Error should contain rate limit information (may be wrapped after retries)
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

// TestCreateOrUpdateUser_ServerError tests ES unavailable (5xx)
func TestCreateOrUpdateUser_ServerError(t *testing.T) {
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	req := &UserRequest{
		Username: "testuser",
		Password: "testpassword",
		Roles:    []string{"admin"},
	}

	err = client.CreateOrUpdateUser(ctx, req)
	if err == nil {
		t.Error("CreateOrUpdateUser() expected error, got nil")
	}

	// Error should contain server error information (may be wrapped after retries)
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

// TestCreateOrUpdateUser_Timeout tests network timeout
func TestCreateOrUpdateUser_Timeout(t *testing.T) {
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       100 * time.Millisecond, // Very short timeout
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	req := &UserRequest{
		Username: "testuser",
		Password: "testpassword",
		Roles:    []string{"admin"},
	}

	err = client.CreateOrUpdateUser(ctx, req)
	if err == nil {
		t.Error("CreateOrUpdateUser() expected timeout error, got nil")
	}
}

// TestCreateOrUpdateUser_RetryLogic tests retry logic with eventual success
func TestCreateOrUpdateUser_RetryLogic(t *testing.T) {
	attemptCount := 0
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// Fail first 2 attempts
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "temporary error"}`))
		} else {
			// Succeed on 3rd attempt
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"created": true}`))
		}
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	req := &UserRequest{
		Username: "testuser",
		Password: "testpassword",
		Roles:    []string{"admin"},
	}

	err = client.CreateOrUpdateUser(ctx, req)
	if err != nil {
		t.Errorf("CreateOrUpdateUser() unexpected error after retries: %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

// TestCreateOrUpdateUser_RetryLogic_AllFail tests retry logic when all attempts fail
func TestCreateOrUpdateUser_RetryLogic_AllFail(t *testing.T) {
	attemptCount := 0
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "persistent error"}`))
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	req := &UserRequest{
		Username: "testuser",
		Password: "testpassword",
		Roles:    []string{"admin"},
	}

	err = client.CreateOrUpdateUser(ctx, req)
	if err == nil {
		t.Error("CreateOrUpdateUser() expected error after all retries, got nil")
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}
}

// TestCreateOrUpdateUser_NoRetryOnAuthError tests that auth errors don't trigger retries
func TestCreateOrUpdateUser_NoRetryOnAuthError(t *testing.T) {
	attemptCount := 0
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "wrongpassword",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	req := &UserRequest{
		Username: "testuser",
		Password: "testpassword",
		Roles:    []string{"admin"},
	}

	err = client.CreateOrUpdateUser(ctx, req)
	if err == nil {
		t.Error("CreateOrUpdateUser() expected error, got nil")
	}

	// Should only attempt once (no retries for auth errors)
	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt (no retries for auth errors), got %d", attemptCount)
	}
}

// TestGetUser_Success tests successful user retrieval
func TestGetUser_Success(t *testing.T) {
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"testuser": map[string]interface{}{
				"username":  "testuser",
				"roles":     []string{"admin", "kibana_user"},
				"full_name": "Test User",
				"email":     "test@example.com",
				"enabled":   true,
				"metadata": map[string]interface{}{
					"source": "test",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	user, err := client.GetUser(ctx, "testuser")
	if err != nil {
		t.Errorf("GetUser() unexpected error: %v", err)
	}

	if user == nil {
		t.Fatal("GetUser() returned nil user")
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user.Username)
	}
	if len(user.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(user.Roles))
	}
	if user.FullName != "Test User" {
		t.Errorf("Expected full name 'Test User', got '%s'", user.FullName)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", user.Email)
	}
	if !user.Enabled {
		t.Error("Expected user to be enabled")
	}
}

// TestGetUser_NotFound tests user not found (404)
func TestGetUser_NotFound(t *testing.T) {
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "user not found"}`))
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	user, err := client.GetUser(ctx, "nonexistent")
	if err != nil {
		t.Errorf("GetUser() unexpected error: %v", err)
	}

	if user != nil {
		t.Error("GetUser() expected nil user for 404, got non-nil")
	}
}

// TestDeleteUser_Success tests successful user deletion
func TestDeleteUser_Success(t *testing.T) {
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"found": true}`))
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	err = client.DeleteUser(ctx, "testuser")
	if err != nil {
		t.Errorf("DeleteUser() unexpected error: %v", err)
	}
}

// TestDeleteUser_NotFound tests deleting non-existent user
func TestDeleteUser_NotFound(t *testing.T) {
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"found": false}`))
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	err = client.DeleteUser(ctx, "nonexistent")
	if err != nil {
		t.Errorf("DeleteUser() unexpected error for 404: %v", err)
	}
}

// TestValidateConnection_Success tests successful connection validation
func TestValidateConnection_Success(t *testing.T) {
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_security/_authenticate" {
			t.Errorf("Expected path /_security/_authenticate, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"username": "admin",
			"roles":    []string{"superuser"},
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	err = client.ValidateConnection(ctx)
	if err != nil {
		t.Errorf("ValidateConnection() unexpected error: %v", err)
	}
}

// TestValidateConnection_Unauthorized tests connection validation with invalid credentials
func TestValidateConnection_Unauthorized(t *testing.T) {
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	})
	defer server.Close()

	client, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "wrongpassword",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	err = client.ValidateConnection(ctx)
	if err == nil {
		t.Error("ValidateConnection() expected error, got nil")
	}

	authErr, ok := err.(*AuthError)
	if !ok {
		t.Errorf("Expected AuthError, got %T", err)
	} else {
		if authErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, authErr.StatusCode)
		}
	}
}

// TestCircuitBreaker_OpensAfterFailures tests that circuit breaker opens after max failures
func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	failureCount := 0
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		failureCount++
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "server error"}`))
	})
	defer server.Close()

	baseClient, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Wrap with circuit breaker
	client := WithCircuitBreaker(baseClient)

	ctx := context.Background()
	req := &UserRequest{
		Username: "testuser",
		Password: "testpassword",
		Roles:    []string{"admin"},
	}

	// Make requests until circuit breaker opens
	// Circuit breaker opens after 5 failures (maxFailures = 5)
	for i := 0; i < 6; i++ {
		err := client.CreateOrUpdateUser(ctx, req)
		if i < 5 {
			// First 5 attempts should reach the server
			if err == nil {
				t.Errorf("Attempt %d: expected error, got nil", i+1)
			}
		} else {
			// 6th attempt should be blocked by circuit breaker
			if err == nil {
				t.Error("Expected circuit breaker error, got nil")
			}
			cbErr, ok := err.(*CircuitBreakerError)
			if !ok {
				t.Errorf("Expected CircuitBreakerError, got %T", err)
			} else if cbErr.State != StateOpen {
				t.Errorf("Expected circuit breaker state Open, got %d", cbErr.State)
			}
		}
	}

	// Verify that only 5 requests reached the server (circuit opened after 5 failures)
	// Note: Each CreateOrUpdateUser retries 3 times, so 5 calls = 15 server requests
	if failureCount != 15 {
		t.Errorf("Expected 15 server requests (5 calls × 3 retries), got %d", failureCount)
	}
}

// TestCircuitBreaker_HalfOpenAfterTimeout tests circuit breaker transitions to half-open
func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	breaker := NewCircuitBreaker()

	// Force circuit to open
	for i := 0; i < 5; i++ {
		breaker.afterCall(fmt.Errorf("test error"))
	}

	if breaker.State() != StateOpen {
		t.Errorf("Expected circuit state Open, got %d", breaker.State())
	}

	// Wait for timeout (circuit breaker timeout is 30s, but we can manipulate time for testing)
	// For this test, we'll manually set the lastStateTime to simulate timeout
	breaker.mu.Lock()
	breaker.lastStateTime = time.Now().Add(-31 * time.Second)
	breaker.mu.Unlock()

	// Next call should transition to half-open
	err := breaker.beforeCall()
	if err != nil {
		t.Errorf("beforeCall() unexpected error: %v", err)
	}

	if breaker.State() != StateHalfOpen {
		t.Errorf("Expected circuit state HalfOpen, got %d", breaker.State())
	}
}

// TestCircuitBreaker_ClosesAfterSuccesses tests circuit breaker closes after successful calls in half-open
func TestCircuitBreaker_ClosesAfterSuccesses(t *testing.T) {
	breaker := NewCircuitBreaker()

	// Force circuit to half-open
	breaker.mu.Lock()
	breaker.state = StateHalfOpen
	breaker.mu.Unlock()

	// Record 2 successes (halfOpenMax = 2)
	breaker.afterCall(nil)
	if breaker.State() != StateHalfOpen {
		t.Errorf("After 1 success: expected HalfOpen, got %d", breaker.State())
	}

	breaker.afterCall(nil)
	if breaker.State() != StateClosed {
		t.Errorf("After 2 successes: expected Closed, got %d", breaker.State())
	}
}

// TestCircuitBreaker_ReopensOnFailureInHalfOpen tests circuit reopens on failure in half-open state
func TestCircuitBreaker_ReopensOnFailureInHalfOpen(t *testing.T) {
	breaker := NewCircuitBreaker()

	// Force circuit to half-open
	breaker.mu.Lock()
	breaker.state = StateHalfOpen
	breaker.mu.Unlock()

	// Record a failure
	breaker.afterCall(fmt.Errorf("test error"))

	if breaker.State() != StateOpen {
		t.Errorf("Expected circuit state Open after failure in HalfOpen, got %d", breaker.State())
	}
}

// TestCircuitBreaker_Reset tests circuit breaker reset
func TestCircuitBreaker_Reset(t *testing.T) {
	breaker := NewCircuitBreaker()

	// Force circuit to open
	for i := 0; i < 5; i++ {
		breaker.afterCall(fmt.Errorf("test error"))
	}

	if breaker.State() != StateOpen {
		t.Errorf("Expected circuit state Open, got %d", breaker.State())
	}

	// Reset
	breaker.Reset()

	if breaker.State() != StateClosed {
		t.Errorf("Expected circuit state Closed after reset, got %d", breaker.State())
	}
}

// TestCircuitBreaker_IntegrationWithClient tests circuit breaker with actual client
func TestCircuitBreaker_IntegrationWithClient(t *testing.T) {
	callCount := 0
	server := mockESServer(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount <= 15 { // First 5 calls × 3 retries = 15 requests
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "server error"}`))
		} else {
			// After circuit opens, no more requests should reach here
			t.Error("Request reached server after circuit breaker should have opened")
			w.WriteHeader(http.StatusOK)
		}
	})
	defer server.Close()

	baseClient, err := NewClient(Config{
		URL:           server.URL,
		AdminUser:     "admin",
		AdminPassword: "password",
		Timeout:       5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client := WithCircuitBreaker(baseClient)

	ctx := context.Background()
	req := &UserRequest{
		Username: "testuser",
		Password: "testpassword",
		Roles:    []string{"admin"},
	}

	// Make 10 requests - circuit should open after 5 failures
	for i := 0; i < 10; i++ {
		err := client.CreateOrUpdateUser(ctx, req)
		if err == nil {
			t.Errorf("Request %d: expected error, got nil", i+1)
		}
	}

	// Verify circuit breaker blocked some requests
	if callCount > 15 {
		t.Errorf("Expected at most 15 server calls (circuit should have opened), got %d", callCount)
	}
}
