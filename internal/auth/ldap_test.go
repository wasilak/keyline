package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
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
		BindPassword: "${LDAP_BIND_PASSWORD}",
		SearchBase:   "DC=example,DC=com",
		SearchFilter: "(sAMAccountName={username})",
	}
}

// userSearchResult returns a minimal single-entry LDAP search result for the user search step.
func userSearchResult(dn, email, displayName string) *ldap.SearchResult {
	entry := ldap.NewEntry(dn, map[string][]string{
		"mail":           {email},
		"displayName":    {displayName},
		"sAMAccountName": {"jdoe"},
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
func newProviderWithMock(t testing.TB, cfg *config.LDAPConfig, conn ldapConn) *LDAPProvider {
	t.Helper()
	// If BindPassword is an env var reference like ${VAR}, set a test value for it so
	// NewLDAPProvider can resolve it during construction.
	if strings.HasPrefix(cfg.BindPassword, "${") && strings.HasSuffix(cfg.BindPassword, "}") {
		envVar := cfg.BindPassword[2 : len(cfg.BindPassword)-1]
		t.Setenv(envVar, "svcpass")
	}

	p, err := NewLDAPProvider(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create LDAPProvider in test helper: %v", err))
	}
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

	// Ensure the environment variable referenced by BindPassword is set for this test.
	// validLDAPConfig() sets BindPassword to ${LDAP_BIND_PASSWORD}.
	t.Setenv("LDAP_BIND_PASSWORD", "svcpass")

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

	p := newProviderWithMock(t, cfg, mock)
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

	p := newProviderWithMock(t, cfg, mock)
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

	p := newProviderWithMock(t, cfg, mock)
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

	p := newProviderWithMock(t, cfg, mock)
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

	p := newProviderWithMock(t, cfg, mock)
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

	p := newProviderWithMock(t, cfg, mock)
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

	p := newProviderWithMock(t, cfg, mock)
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

	p := newProviderWithMock(t, cfg, mock)
	p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: basicAuthHeader(maliciousUsername, "pass"),
	})

	// The filter must not contain the raw special characters unescaped.
	assert.NotContains(t, capturedFilter, ")(cn=*",
		"LDAP filter must escape special characters to prevent injection")
}

func TestMappedUsernameNormalization(t *testing.T) {
	cfg := validLDAPConfig()
	// override attribute mapping to ensure mapping via config works
	cfg.AttributeMapping = map[string]string{"username": "sAMAccountName", "email": "mail", "displayName": "displayName"}

	userDN := "CN=Strange Name,DC=example,DC=com"

	mock := &mockLDAPConn{
		bindFn: func(_, _ string) error { return nil },
		searchFn: func(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
			if req.BaseDN == cfg.SearchBase {
				// username attribute contains spaces and capitals and illegal chars
				entry := ldap.NewEntry(userDN, map[string][]string{
					"sAMAccountName": {"  John.Doe@Example ", "JOHN"},
					"mail":           {"john@example.com"},
					"displayName":    {"John Doe"},
				})
				return &ldap.SearchResult{Entries: []*ldap.Entry{entry}}, nil
			}
			return groupSearchResult(), nil
		},
	}

	p := newProviderWithMock(t, cfg, mock)
	res := p.Authenticate(context.Background(), &AuthRequest{AuthorizationHeader: basicAuthHeader("jdoe", "pass")})

	require.True(t, res.Authenticated)
	// normalized username: "john.doe_example" => lowercased, truncate invalid chars replaced with underscore
	assert.Equal(t, "john.doe_example", res.Username)
}

func TestMappedUsernameMissingFallsBack(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.AttributeMapping = map[string]string{"username": "sAMAccountName", "email": "mail", "displayName": "displayName"}

	userDN := "CN=NoLogin,DC=example,DC=com"

	mock := &mockLDAPConn{
		bindFn: func(_, _ string) error { return nil },
		searchFn: func(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
			if req.BaseDN == cfg.SearchBase {
				// no username attribute present
				entry := ldap.NewEntry(userDN, map[string][]string{
					"mail":        {"nologin@example.com"},
					"displayName": {"No Login"},
				})
				return &ldap.SearchResult{Entries: []*ldap.Entry{entry}}, nil
			}
			return groupSearchResult(), nil
		},
	}

	p := newProviderWithMock(t, cfg, mock)
	res := p.Authenticate(context.Background(), &AuthRequest{AuthorizationHeader: basicAuthHeader("fallback", "pass")})

	require.True(t, res.Authenticated)
	assert.Equal(t, "fallback", res.Username)
}

// --- Constructor edge cases ---

func TestNewLDAPProvider_InlineBindPasswordRejected(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.BindPassword = "plaintext-secret"

	p, err := NewLDAPProvider(cfg)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "environment variable reference")
}

func TestNewLDAPProvider_BindPasswordEnvVarNotSet(t *testing.T) {
	os.Unsetenv("LDAP_MISSING_VAR")
	cfg := validLDAPConfig()
	cfg.BindPassword = "${LDAP_MISSING_VAR}"

	p, err := NewLDAPProvider(cfg)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "LDAP_MISSING_VAR")
}

func TestNewLDAPProvider_BindPasswordEmptyEnvVarName(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.BindPassword = "${}"

	p, err := NewLDAPProvider(cfg)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "empty")
}

// --- Authenticate edge cases ---

func TestLDAPProvider_Authenticate_MissingHeader(t *testing.T) {
	cfg := validLDAPConfig()
	t.Setenv("LDAP_BIND_PASSWORD", "svcpass")

	p, err := NewLDAPProvider(cfg)
	require.NoError(t, err)

	result := p.Authenticate(context.Background(), &AuthRequest{AuthorizationHeader: ""})

	assert.False(t, result.Authenticated)
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "missing Authorization header")
}

func TestLDAPProvider_Authenticate_NonBasicHeader(t *testing.T) {
	cfg := validLDAPConfig()
	t.Setenv("LDAP_BIND_PASSWORD", "svcpass")

	p, err := NewLDAPProvider(cfg)
	require.NoError(t, err)

	result := p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: "Bearer sometoken",
	})

	assert.False(t, result.Authenticated)
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "not Basic auth")
}

func TestLDAPProvider_Authenticate_InvalidBase64(t *testing.T) {
	cfg := validLDAPConfig()
	t.Setenv("LDAP_BIND_PASSWORD", "svcpass")

	p, err := NewLDAPProvider(cfg)
	require.NoError(t, err)

	result := p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: "Basic not-valid-base64!!!",
	})

	assert.False(t, result.Authenticated)
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "invalid base64")
}

func TestLDAPProvider_Authenticate_DialFailure(t *testing.T) {
	cfg := validLDAPConfig()
	t.Setenv("LDAP_BIND_PASSWORD", "svcpass")

	p, err := NewLDAPProvider(cfg)
	require.NoError(t, err)
	p.dialFn = func(_ *config.LDAPConfig) (ldapConn, error) {
		return nil, fmt.Errorf("connection refused")
	}

	result := p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: basicAuthHeader("jdoe", "pass"),
	})

	assert.False(t, result.Authenticated)
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "LDAP connection failed")
}

func TestLDAPProvider_Authenticate_ServiceAccountRebindFail(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.GroupSearchBase = "OU=groups,DC=example,DC=com"
	cfg.GroupSearchFilter = "(member={user_dn})"

	userDN := "CN=jdoe,DC=example,DC=com"
	bindCall := 0

	mock := &mockLDAPConn{
		bindFn: func(_, _ string) error {
			bindCall++
			if bindCall == 3 { // third bind = service account re-bind after user bind
				return fmt.Errorf("re-bind failed")
			}
			return nil
		},
		searchFn: func(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
			return userSearchResult(userDN, "jdoe@example.com", "John Doe"), nil
		},
	}

	p := newProviderWithMock(t, cfg, mock)
	result := p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: basicAuthHeader("jdoe", "pass"),
	})

	assert.False(t, result.Authenticated)
	require.NotNil(t, result.Error)
	assert.Contains(t, result.Error.Error(), "LDAP service unavailable")
}

// --- Group search filter substitution ---

func TestLDAPProvider_GroupSearchFilter_UserDNSubstituted(t *testing.T) {
	cfg := validLDAPConfig()
	cfg.GroupSearchBase = "OU=groups,DC=example,DC=com"
	cfg.GroupSearchFilter = "(member={user_dn})"

	userDN := "CN=jdoe,OU=people,DC=example,DC=com"
	var capturedGroupFilter string

	mock := &mockLDAPConn{
		bindFn: func(_, _ string) error { return nil },
		searchFn: func(req *ldap.SearchRequest) (*ldap.SearchResult, error) {
			if req.BaseDN == cfg.GroupSearchBase {
				capturedGroupFilter = req.Filter
				return groupSearchResult(), nil
			}
			return userSearchResult(userDN, "jdoe@example.com", "John Doe"), nil
		},
	}

	p := newProviderWithMock(t, cfg, mock)
	p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: basicAuthHeader("jdoe", "pass"),
	})

	// {user_dn} must be replaced with the actual (escaped) user DN — not the literal placeholder.
	assert.NotContains(t, capturedGroupFilter, "{user_dn}")
	assert.Contains(t, capturedGroupFilter, "CN=jdoe")
}

func TestLDAPProvider_GroupSearch_NoBaseConfigured_ReturnsEmpty(t *testing.T) {
	cfg := validLDAPConfig()
	// No GroupSearchBase / GroupSearchFilter set.

	userDN := "CN=jdoe,DC=example,DC=com"
	searchCalls := 0

	mock := &mockLDAPConn{
		bindFn: func(_, _ string) error { return nil },
		searchFn: func(_ *ldap.SearchRequest) (*ldap.SearchResult, error) {
			searchCalls++
			return userSearchResult(userDN, "jdoe@example.com", "John Doe"), nil
		},
	}

	p := newProviderWithMock(t, cfg, mock)
	result := p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: basicAuthHeader("jdoe", "pass"),
	})

	require.True(t, result.Authenticated)
	assert.Empty(t, result.Groups)
	// Only one search should happen (user search); group search must be skipped.
	assert.Equal(t, 1, searchCalls)
}

func TestLDAPProvider_GroupSearch_EmptyResult(t *testing.T) {
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
			return &ldap.SearchResult{}, nil // user is in no groups
		},
	}

	p := newProviderWithMock(t, cfg, mock)
	result := p.Authenticate(context.Background(), &AuthRequest{
		AuthorizationHeader: basicAuthHeader("jdoe", "pass"),
	})

	require.True(t, result.Authenticated)
	assert.Empty(t, result.Groups)
}

// --- dialLDAP TLS mode tests ---
// These tests call the real dialLDAP against an unreachable address to verify
// each TLS branch executes and surfaces the expected error prefix.

func TestDialLDAP_PlaintextMode(t *testing.T) {
	cfg := &config.LDAPConfig{URL: "ldap://127.0.0.1:19389", TLSMode: ""}
	_, err := dialLDAP(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LDAP dial failed")
}

func TestDialLDAP_NoneMode(t *testing.T) {
	cfg := &config.LDAPConfig{URL: "ldap://127.0.0.1:19389", TLSMode: "none"}
	_, err := dialLDAP(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LDAP dial failed")
}

func TestDialLDAP_LDAPSMode(t *testing.T) {
	cfg := &config.LDAPConfig{URL: "ldaps://127.0.0.1:19636", TLSMode: "ldaps", TLSSkipVerify: true}
	_, err := dialLDAP(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LDAPS dial failed")
}

func TestDialLDAP_StartTLSMode(t *testing.T) {
	cfg := &config.LDAPConfig{URL: "ldap://127.0.0.1:19389", TLSMode: "starttls"}
	_, err := dialLDAP(cfg)
	require.Error(t, err)
	// starttls first dials plain then negotiates — connection refused hits the plain dial.
	assert.Contains(t, err.Error(), "LDAP dial failed")
}

func TestDialLDAP_TLSSkipVerify_PropagatedToConfig(t *testing.T) {
	// TLSSkipVerify=true with ldaps — error should come from LDAPS dial (not TLS verify),
	// confirming the TLS config was constructed and passed to DialURL.
	cfg := &config.LDAPConfig{URL: "ldaps://127.0.0.1:19636", TLSMode: "ldaps", TLSSkipVerify: true}
	_, err := dialLDAP(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LDAPS dial failed")
}

// --- normalizeUsername table-driven tests ---

func TestNormalizeUsername(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"jdoe", "jdoe"},
		{"  jdoe  ", "jdoe"},
		{"JDOE", "jdoe"},
		{"john.doe", "john.doe"},
		{"john-doe", "john-doe"},
		{"john_doe", "john_doe"},
		{"john@example.com", "john_example.com"},
		{"  John.Doe@Example ", "john.doe_example"},
		{"!!!bad!!!", "bad"},
		{"", ""},
		{"@@@", ""},
		{"a@@b", "a_b"},
		{"__leading", "leading"},
		{"trailing__", "trailing"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			assert.Equal(t, tc.want, normalizeUsername(tc.input))
		})
	}
}

// --- hasAnyGroup table-driven tests ---

func TestHasAnyGroup(t *testing.T) {
	cases := []struct {
		userGroups     []string
		requiredGroups []string
		want           bool
	}{
		{[]string{"admins", "users"}, []string{"admins"}, true},
		{[]string{"users"}, []string{"admins"}, false},
		{[]string{}, []string{"admins"}, false},
		{[]string{"admins"}, []string{}, false},
		{[]string{}, []string{}, false},
		{[]string{"a", "b", "c"}, []string{"x", "b"}, true},
	}

	for _, tc := range cases {
		got := hasAnyGroup(tc.userGroups, tc.requiredGroups)
		assert.Equal(t, tc.want, got, "userGroups=%v requiredGroups=%v", tc.userGroups, tc.requiredGroups)
	}
}
