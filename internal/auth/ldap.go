package auth

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/wasilak/keyline/internal/config"
)

const (
	ldapDefaultConnectionTimeout    = 10 * time.Second
	ldapDefaultUsernameAttribute    = "sAMAccountName"
	ldapDefaultEmailAttribute       = "mail"
	ldapDefaultDisplayNameAttribute = "displayName"
	ldapDefaultGroupNameAttribute   = "cn"
)

// ldapConn abstracts *ldap.Conn for testability.
type ldapConn interface {
	Bind(username, password string) error
	Search(searchRequest *ldap.SearchRequest) (*ldap.SearchResult, error)
	SetTimeout(timeout time.Duration)
	Close() error
}

// LDAPProvider implements Basic Auth via an LDAP/Active Directory server.
type LDAPProvider struct {
	config *config.LDAPConfig
	// dialFn is swapped in tests to inject a mock connection.
	dialFn func(cfg *config.LDAPConfig) (ldapConn, error)
}

// NewLDAPProvider creates a new LDAPProvider from the given config.
// Returns an error if the config is disabled or missing required fields.
func NewLDAPProvider(cfg *config.LDAPConfig) (*LDAPProvider, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("LDAP authentication is not enabled")
	}

	if cfg.URL == "" {
		return nil, fmt.Errorf("ldap.url is required")
	}

	// Capture explicitly configured attribute values before defaults are applied.
	origUsername := cfg.UsernameAttribute
	origEmail := cfg.EmailAttribute
	origDisplayName := cfg.DisplayNameAttribute
	origGroupName := cfg.GroupNameAttribute

	// Apply defaults for optional attribute names.
	if cfg.UsernameAttribute == "" {
		cfg.UsernameAttribute = ldapDefaultUsernameAttribute
	}
	if cfg.EmailAttribute == "" {
		cfg.EmailAttribute = ldapDefaultEmailAttribute
	}
	if cfg.DisplayNameAttribute == "" {
		cfg.DisplayNameAttribute = ldapDefaultDisplayNameAttribute
	}
	if cfg.GroupNameAttribute == "" {
		cfg.GroupNameAttribute = ldapDefaultGroupNameAttribute
	}
	if cfg.ConnectionTimeout == 0 {
		cfg.ConnectionTimeout = ldapDefaultConnectionTimeout
	}

	// Apply attribute_mapping overrides, warning when they shadow an explicit field.
	if cfg.AttributeMapping != nil {
		if v, ok := cfg.AttributeMapping["username"]; ok && v != "" {
			if origUsername != "" {
				slog.Info("attribute_mapping.username overrides username_attribute — using mapped value",
					slog.String("original", origUsername), slog.String("mapped", v))
			}
			cfg.UsernameAttribute = v
		}
		if v, ok := cfg.AttributeMapping["email"]; ok && v != "" {
			if origEmail != "" {
				slog.Info("attribute_mapping.email overrides email_attribute — using mapped value",
					slog.String("original", origEmail), slog.String("mapped", v))
			}
			cfg.EmailAttribute = v
		}
		if v, ok := cfg.AttributeMapping["displayName"]; ok && v != "" {
			if origDisplayName != "" {
				slog.Info("attribute_mapping.displayName overrides display_name_attribute — using mapped value",
					slog.String("original", origDisplayName), slog.String("mapped", v))
			}
			cfg.DisplayNameAttribute = v
		}
		if v, ok := cfg.AttributeMapping["groupName"]; ok && v != "" {
			if origGroupName != "" {
				slog.Info("attribute_mapping.groupName overrides group_name_attribute — using mapped value",
					slog.String("original", origGroupName), slog.String("mapped", v))
			}
			cfg.GroupNameAttribute = v
		}
	}

	// Warn when credentials will be transmitted in plaintext (#29).
	if cfg.TLSMode == "" || cfg.TLSMode == "none" {
		slog.Warn("LDAP TLS mode is 'none' — credentials will be transmitted in plaintext. Use 'ldaps' or 'starttls' in production.")
	}

	// Resolve bind password from environment variable reference. Expect form ${ENV_VAR}.
	if cfg.BindPassword == "" {
		return nil, fmt.Errorf("ldap.bind_password is required")
	}
	if strings.HasPrefix(cfg.BindPassword, "${") && strings.HasSuffix(cfg.BindPassword, "}") {
		envVar := cfg.BindPassword[2 : len(cfg.BindPassword)-1]
		if envVar == "" {
			return nil, fmt.Errorf("ldap.bind_password environment variable name is empty")
		}
		val := os.Getenv(envVar)
		if val == "" {
			return nil, fmt.Errorf("ldap.bind_password environment variable %s is not set", envVar)
		}
		cfg.BindPassword = val
	} else {
		// Enforce env var format — project is greenfield.
		return nil, fmt.Errorf("ldap.bind_password must be an environment variable reference in the form ${ENV_VAR}")
	}

	return &LDAPProvider{
		config: cfg,
		dialFn: dialLDAP,
	}, nil
}

// Authenticate validates Basic Auth credentials against LDAP.
func (p *LDAPProvider) Authenticate(ctx context.Context, req *AuthRequest) *AuthResult {
	slog.InfoContext(ctx, "Attempting LDAP authentication")

	// --- 1. Extract credentials from the Basic Auth header ---
	if req.AuthorizationHeader == "" {
		return &AuthResult{Authenticated: false, Error: fmt.Errorf("missing Authorization header")}
	}
	if !strings.HasPrefix(req.AuthorizationHeader, "Basic ") {
		return &AuthResult{Authenticated: false, Error: fmt.Errorf("not Basic auth")}
	}

	encodedCreds := strings.TrimPrefix(req.AuthorizationHeader, "Basic ")
	if encodedCreds == "" {
		return &AuthResult{Authenticated: false, Error: fmt.Errorf("empty credentials")}
	}

	decodedBytes, err := base64.StdEncoding.DecodeString(encodedCreds)
	if err != nil {
		slog.WarnContext(ctx, "Failed to decode LDAP auth header", slog.String("error", err.Error()))
		return &AuthResult{Authenticated: false, Error: fmt.Errorf("invalid base64 encoding")}
	}

	username, password, err := extractCredentials(string(decodedBytes))
	if err != nil {
		slog.WarnContext(ctx, "Failed to extract LDAP credentials", slog.String("error", err.Error()))
		return &AuthResult{Authenticated: false, Error: err}
	}

	// Escape the username to prevent LDAP injection.
	safeUsername := ldap.EscapeFilter(username)

	// --- 2. Connect to LDAP server ---
	conn, err := p.dialFn(p.config)
	if err != nil {
		slog.ErrorContext(ctx, "LDAP connection failed", slog.String("error", err.Error()))
		return &AuthResult{Authenticated: false, Error: fmt.Errorf("LDAP connection failed")}
	}
	defer conn.Close() //nolint:errcheck

	conn.SetTimeout(p.config.ConnectionTimeout)

	// --- 3. Service account bind ---
	if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
		slog.ErrorContext(ctx, "LDAP service account bind failed", slog.String("error", err.Error()))
		return &AuthResult{Authenticated: false, Error: fmt.Errorf("LDAP service unavailable")}
	}

	// --- 4. Search for the user ---
	userDN, mappedUsername, email, displayName, err := p.searchUser(conn, safeUsername)
	if err != nil {
		slog.WarnContext(ctx, "LDAP user search failed",
			slog.String("username", username),
			slog.String("error", err.Error()),
		)
		return &AuthResult{Authenticated: false, Username: username, Error: err}
	}

	// --- 5. Bind as the user (password verification) ---
	if err := conn.Bind(userDN, password); err != nil {
		slog.WarnContext(ctx, "LDAP user bind failed (wrong password)",
			slog.String("username", username),
		)
		return &AuthResult{Authenticated: false, Username: username, Error: fmt.Errorf("invalid credentials")}
	}

	// --- 6. Re-bind as service account for group search ---
	if err := conn.Bind(p.config.BindDN, p.config.BindPassword); err != nil {
		slog.ErrorContext(ctx, "LDAP service account re-bind failed", slog.String("error", err.Error()))
		return &AuthResult{Authenticated: false, Error: fmt.Errorf("LDAP service unavailable")}
	}

	// --- 7. Search for groups (non-fatal) ---
	groups := p.searchGroups(ctx, conn, userDN)

	// --- 8. Check required groups ---
	if len(p.config.RequiredGroups) > 0 && !hasAnyGroup(groups, p.config.RequiredGroups) {
		slog.WarnContext(ctx, "LDAP user not in required groups",
			slog.String("username", username),
			slog.Any("required", p.config.RequiredGroups),
			slog.Any("actual", groups),
		)
		return &AuthResult{Authenticated: false, Username: username, Error: fmt.Errorf("user not in required groups")}
	}

	slog.InfoContext(ctx, "LDAP authentication successful",
		slog.String("username", username),
		slog.Any("groups", groups),
	)

	// Use mappedUsername (from LDAP attribute) when available; fall back to provided username.
	finalUsername := username
	if mappedUsername != "" {
		if n := normalizeUsername(mappedUsername); n != "" {
			finalUsername = n
		}
	}

	return &AuthResult{
		Authenticated: true,
		Username:      finalUsername,
		Email:         email,
		FullName:      displayName,
		Groups:        groups,
		Source:        "ldap",
	}
}

// searchUser performs an LDAP search to find the user DN and attributes.
// Returns the user's DN, email, display name, and any error.
func (p *LDAPProvider) searchUser(conn ldapConn, safeUsername string) (userDN, mappedUsername, email, displayName string, err error) {
	filter := strings.ReplaceAll(p.config.SearchFilter, "{username}", safeUsername)

	req := ldap.NewSearchRequest(
		p.config.SearchBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, // size limit — we only need one result
		0,
		false,
		filter,
		[]string{
			p.config.UsernameAttribute,
			p.config.EmailAttribute,
			p.config.DisplayNameAttribute,
		},
		nil,
	)

	result, err := conn.Search(req)
	if err != nil {
		return "", "", "", "", fmt.Errorf("LDAP user search error: %w", err)
	}

	if len(result.Entries) == 0 {
		return "", "", "", "", fmt.Errorf("user not found")
	}

	entry := result.Entries[0]
	return entry.DN,
		entry.GetAttributeValue(p.config.UsernameAttribute),
		entry.GetAttributeValue(p.config.EmailAttribute),
		entry.GetAttributeValue(p.config.DisplayNameAttribute),
		nil
}

// searchGroups performs an optional LDAP search for group memberships.
// Failure is non-fatal: returns empty slice and logs a warning.
func (p *LDAPProvider) searchGroups(ctx context.Context, conn ldapConn, userDN string) []string {
	if p.config.GroupSearchBase == "" || p.config.GroupSearchFilter == "" {
		return []string{}
	}

	filter := strings.ReplaceAll(p.config.GroupSearchFilter, "{user_dn}", ldap.EscapeFilter(userDN))

	req := ldap.NewSearchRequest(
		p.config.GroupSearchBase,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		0,
		false,
		filter,
		[]string{p.config.GroupNameAttribute},
		nil,
	)

	result, err := conn.Search(req)
	if err != nil {
		slog.WarnContext(ctx, "LDAP group search failed (non-fatal), continuing with empty groups",
			slog.String("error", err.Error()),
		)
		return []string{}
	}

	groups := make([]string, 0, len(result.Entries))
	for _, entry := range result.Entries {
		if name := entry.GetAttributeValue(p.config.GroupNameAttribute); name != "" {
			groups = append(groups, name)
		}
	}

	return groups
}

// dialLDAP opens a connection to the LDAP server using the configured TLS mode.
func dialLDAP(cfg *config.LDAPConfig) (ldapConn, error) {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: cfg.TLSSkipVerify, //nolint:gosec // controlled by operator config
	}

	switch cfg.TLSMode {
	case "ldaps":
		conn, err := ldap.DialURL(cfg.URL, ldap.DialWithTLSConfig(tlsCfg))
		if err != nil {
			return nil, fmt.Errorf("LDAPS dial failed: %w", err)
		}
		return conn, nil

	case "starttls":
		conn, err := ldap.DialURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("LDAP dial failed: %w", err)
		}
		if err := conn.StartTLS(tlsCfg); err != nil {
			conn.Close() //nolint:errcheck
			return nil, fmt.Errorf("StartTLS failed: %w", err)
		}
		return conn, nil

	default: // "none" or ""
		conn, err := ldap.DialURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("LDAP dial failed: %w", err)
		}
		return conn, nil
	}
}

// hasAnyGroup returns true if the user's groups contain at least one required group.
func hasAnyGroup(userGroups, requiredGroups []string) bool {
	groupSet := make(map[string]struct{}, len(userGroups))
	for _, g := range userGroups {
		groupSet[g] = struct{}{}
	}
	for _, required := range requiredGroups {
		if _, ok := groupSet[required]; ok {
			return true
		}
	}
	return false
}

// normalizeUsername converts an LDAP username attribute into a safe Keyline username.
// Rules:
// - trim spaces
// - lowercase
// - allow: a-z, 0-9, '.', '_', '-'
// - replace other chars with single underscore
// - collapse multiple underscores and trim leading/trailing underscores
func normalizeUsername(u string) string {
	u = strings.TrimSpace(u)
	u = strings.ToLower(u)
	var b strings.Builder
	lastUnderscore := false
	for _, r := range u {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	s := b.String()
	s = strings.Trim(s, "_")
	return s
}
