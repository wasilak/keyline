package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourusername/keyline/internal/config"
)

// --- Mock LDAP connection ---

// mockLDAPConn implements ldapConn for tests.
type mockLDAPConn struct {
	bindFn   func(username, password string) error
	searchFn func(req *ldap.SearchRequest) (*ldap.SearchResult, error)
	closed   bool
}

func (m *mockLDAPConn) Bind(username, password string) error {
	if m.bindFn != nil {
		return m.bindFn(username, password)
	}
	return nil
}

func (m *mockLDAPConn) Search(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
	if m.searchFn != nil {
		return m.searchFn(req)
	}
	return &ldap.SearchResult{}, nil
}

func (m *mockLDAPConn) SetTimeout(_ time.Duration) {}

func (m *mockLDAPConn) Close() error {
	m.closed = true
	return nil
}

// --- Helpers ---

// basicAuthHeader encodes username:password as a Basic Auth header value.
func basicAuthHeader(username, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
}

// validLDAPConfig returns a minimal enabled LDAPConfig for constructing a provider.
func validLDAPConfig() *config.LDAPConfig {
	return &config.LDAPConfig{
		Enabled:      true,
		URL:          "ldap://ldap.example.com:389",
		BindDN:       "CN=svc,DC=example,DC=com",
		BindPassword: "svcpass",
		SearchBase:   "DC=example,DC=com",
		SearchFilter: "(sAMAccountName={username})",
	}
}

// userSearchResult returns a minimal single-entry LDAP search result for the user search step.
func userSearchResult(dn, email, displayName string) *ldap.SearchResult {
	entry := ldap.NewEntry(dn, map[string][]string{
		"mail":        {email},
		"displayName": {displayName},
	})
	return &ldap.SearchResult{Entries: []*ldap.Entry{entry}}
}

// groupSearchResult returns a search result with the given group CNs.
func groupSearchResult(cns ...string) *ldap.SearchResult {
	entries := make([]*ldap.Entry, 0, len(cns))
	for _, cn := range cns {
		entries = append(entries, ldap.NewEntry("CN="+cn+",DC=example,DC=com", map[string][]string{
			"cn": {cn},
		}))
	}
	return &ldap.SearchResult{Entries: entries}
}

// newProviderWithMock creates an LDAPProvider using the given mock connection.
func newProviderWithMock(cfg *config.LDAPConfig, conn ldapConn) *LDAPProvider {
	p, _ := NewLDAPProvider(cfg)
	p.dialFn = func(_ *config.LDAPConfig) (ldapConn, error) {
		return conn, nil
	}
	return p
}

// --- Constructor tests ---

func TestNewLDAPProvider_NotEnabled(t *testing.T) {
	cfg := &config.LDAPConfig{Enabled: false}
	p, err := NewLDAPProvider(cfg)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestNewLDAPProvider_MissingURL(t *testing.T) {
	cfg := &config.LDAPConfig{
		Enabled: true,
		URL:     "",
	}
	p, err := NewLDAPProvider(cfg)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "ldap.url is required")
}

func TestNewLDAPProvider_Defaults(t *testing.T) {
	cfg := validLDAPConfig()
	// Clear optional attribute fields so defaults are applied.
	cfg.UsernameAttribute = ""
	cfg.EmailAttribute = ""
	cfg.DisplayNameAttribute = ""
	cfg.GroupNameAttribute = ""
	cfg.ConnectionTimeout = 0

	p, err := NewLDAPProvider(cfg)
	require.NoError(t, err)
	require.NotNil(t, p)

	assert.Equal(t, ldapDefaultUsernameAttribute, cfg.UsernameAttribute)
	assert.Equal(t, ldapDefaultEmailAttribute, cfg.EmailAttribute)
	assert.Equal(t, ldapDefaultDisplayNameAttribute, cfg.DisplayNameAttribute)
	assert.Equal(t, ldapDefaultGroupNameAttribute, cfg.GroupNameAttribute)
	assert.Equal(t, ldapDefaultConnectionTimeout, cfg.ConnectionTimeout)
}

// --- Authenticate tests ---

func TestLDAPProvider_Authenticate_Success(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.GroupSearchBase = "OU=groups,DC=example,DC=com"
	cfg.GroupSearchFilter = "(member={user_dn})"

	userDN := "CN=jdoe,DC=example,DC=com"
	bindCalls := 0

	mock := &mockLDAPConn{
		bindFn: func(username, _ string) error {
			bindCalls++
			return nil // all binds succeed
		},
		searchFn: func(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
			if req.BaseDN == cfg.SearchBase {
				return userSearchResult(userDN, "jdoe@example.com", "John Doe"), nil
			}
			return groupSearchResult("developers", "users"), nil
		},
	}

	p := newProviderWithMock(cfg, mock)
	result := p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: basicAuthHeader("jdoe", "s3cr3t"),
	})

	require.True(t, result.Authenticated)
	assert.Equal(t, "jdoe", result.Username)
	assert.Equal(t, "jdoe@example.com", result.Email)
	assert.Equal(t, "John Doe", result.FullName)
	assert.Equal(t, []string{"developers", "users"}, result.Groups)
	assert.Equal(t, "ldap", result.Source)
	assert.Nil(t, result.Error)
	assert.Equal(t, 3, bindCalls) // service bind + user bind + re-bind
}

func TestLDAPProvider_Authenticate_WrongPassword(t *testing.T) {
	cfg := validLDAPConfig()
	userDN := "CN=jdoe,DC=example,DC=com"
	bindCall := 0

	mock := &mockLDAPConn{
		bindFn: func(username, _ string) error {
			bindCall++
			if bindCall == 1 {
				return nil // service account bind succeeds
			}
			// User bind (second call) fails — wrong password
			return fmt.Errorf("invalid credentials")
		},
		searchFn: func(_ *ldap.SearchRequest) (*ldap.SearchResult, error) {
			return userSearchResult(userDN, "", ""), nil
		},
	}

	p := newProviderWithMock(cfg, mock)
	result := p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: basicAuthHeader("jdoe", "wrongpass"),
	})

	assert.False(t, result.Authenticated)
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "invalid credentials")
}

func TestLDAPProvider_Authenticate_UserNotFound(t *testing.T) {
	cfg := validLDAPConfig()

	mock := &mockLDAPConn{
		bindFn: func(_, _ string) error { return nil },
		searchFn: func(_ *ldap.SearchRequest) (*ldap.SearchResult, error) {
			return &ldap.SearchResult{Entries: []*ldap.Entry{}}, nil // 0 entries
		},
	}

	p := newProviderWithMock(cfg, mock)
	result := p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: basicAuthHeader("ghost", "pass"),
	})

	assert.False(t, result.Authenticated)
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "user not found")
}

func TestLDAPProvider_Authenticate_ServiceAccountBindFail(t *testing.T) {
	cfg := validLDAPConfig()

	mock := &mockLDAPConn{
		bindFn: func(_, _ string) error {
			return fmt.Errorf("connection refused")
		},
	}

	p := newProviderWithMock(cfg, mock)
	result := p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: basicAuthHeader("jdoe", "pass"),
	})

	assert.False(t, result.Authenticated)
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "LDAP service unavailable")
}

func TestLDAPProvider_Authenticate_GroupSearchFail_ContinuesWithEmptyGroups(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.GroupSearchBase = "OU=groups,DC=example,DC=com"
	cfg.GroupSearchFilter = "(member={user_dn})"

	userDN := "CN=jdoe,DC=example,DC=com"

	mock := &mockLDAPConn{
		bindFn: func(_, _ string) error { return nil },
		searchFn: func(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
			if req.BaseDN == cfg.SearchBase {
				return userSearchResult(userDN, "jdoe@example.com", "John Doe"), nil
			}
			// Group search fails — non-fatal
			return nil, fmt.Errorf("group search timeout")
		},
	}

	p := newProviderWithMock(cfg, mock)
	result := p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: basicAuthHeader("jdoe", "pass"),
	})

	// Auth should still succeed, just with empty groups.
	require.True(t, result.Authenticated)
	assert.Empty(t, result.Groups)
	assert.Nil(t, result.Error)
}

func TestLDAPProvider_Authenticate_RequiredGroupsNotMet(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.GroupSearchBase = "OU=groups,DC=example,DC=com"
	cfg.GroupSearchFilter = "(member={user_dn})"
	cfg.RequiredGroups = []string{"admins"}

	userDN := "CN=jdoe,DC=example,DC=com"

	mock := &mockLDAPConn{
		bindFn: func(_, _ string) error { return nil },
		searchFn: func(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
			if req.BaseDN == cfg.SearchBase {
				return userSearchResult(userDN, "", ""), nil
			}
			return groupSearchResult("viewers"), nil // user only in "viewers", not "admins"
		},
	}

	p := newProviderWithMock(cfg, mock)
	result := p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: basicAuthHeader("jdoe", "pass"),
	})

	assert.False(t, result.Authenticated)
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "required groups")
}

func TestLDAPProvider_Authenticate_RequiredGroupsMet(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.GroupSearchBase = "OU=groups,DC=example,DC=com"
	cfg.GroupSearchFilter = "(member={user_dn})"
	cfg.RequiredGroups = []string{"admins"}

	userDN := "CN=jdoe,DC=example,DC=com"

	mock := &mockLDAPConn{
		bindFn: func(_, _ string) error { return nil },
		searchFn: func(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
			if req.BaseDN == cfg.SearchBase {
				return userSearchResult(userDN, "jdoe@example.com", "John Doe"), nil
			}
			return groupSearchResult("admins", "viewers"), nil
		},
	}

	p := newProviderWithMock(cfg, mock)
	result := p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: basicAuthHeader("jdoe", "pass"),
	})

	require.True(t, result.Authenticated)
	assert.Contains(t, result.Groups, "admins")
	assert.Nil(t, result.Error)
}

func TestLDAPProvider_InjectionPrevention(t *testing.T) {
	cfg := validLDAPConfig()

	var capturedFilter string
	mock := &mockLDAPConn{
		bindFn: func(_, _ string) error { return nil },
		searchFn: func(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
			capturedFilter = req.Filter
			// Return 0 entries so the auth fails cleanly after we've captured the filter.
			return &ldap.SearchResult{Entries: []*ldap.Entry{}}, nil
		},
	}

	// Username containing LDAP special characters that must be escaped.
	maliciousUsername := "jdoe)(cn=*"

	p := newProviderWithMock(cfg, mock)
	p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: basicAuthHeader(maliciousUsername, "pass"),
	})

	// The filter must not contain the raw special characters unescaped.
	assert.NotContains(t, capturedFilter, ")(cn=*",
		"LDAP filter must escape special characters to prevent injection")
}
