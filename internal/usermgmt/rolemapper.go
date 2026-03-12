package usermgmt

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/yourusername/keyline/internal/config"
)

// RoleMapper maps user groups/claims to Elasticsearch roles
type RoleMapper struct {
	config *config.Config
}

// NewRoleMapper creates a new RoleMapper instance
func NewRoleMapper(cfg *config.Config) *RoleMapper {
	return &RoleMapper{
		config: cfg,
	}
}

// MapGroupsToRoles maps user groups to Elasticsearch roles based on role mappings
// Returns the list of ES roles to assign to the user
func (rm *RoleMapper) MapGroupsToRoles(ctx context.Context, groups []string) ([]string, error) {
	rolesSet := make(map[string]bool)
	matched := false

	// Iterate through all role mappings
	for _, mapping := range rm.config.RoleMappings {
		// Match each group against each mapping pattern
		for _, group := range groups {
			if rm.matchPattern(group, mapping.Pattern) {
				matched = true
				// Collect ALL matching roles (map deduplicates automatically)
				for _, role := range mapping.ESRoles {
					rolesSet[role] = true
				}

				// Log matched mappings
				slog.DebugContext(ctx, "Role mapping matched",
					slog.String("group", group),
					slog.String("pattern", mapping.Pattern),
					slog.Any("roles", mapping.ESRoles),
				)
			}
		}
	}

	// If no matches and default_es_roles defined, use defaults
	if !matched {
		if len(rm.config.DefaultESRoles) == 0 {
			// If no matches and no defaults, return error
			return nil, fmt.Errorf("no role mappings matched and no default roles configured")
		}

		slog.InfoContext(ctx, "No role mappings matched, using default roles",
			slog.Any("default_roles", rm.config.DefaultESRoles),
		)

		for _, role := range rm.config.DefaultESRoles {
			rolesSet[role] = true
		}
	}

	// Convert set to slice
	roles := make([]string, 0, len(rolesSet))
	for role := range rolesSet {
		roles = append(roles, role)
	}

	return roles, nil
}

// matchPattern matches a value against a pattern with wildcard support
// Supports:
// - Exact match: "admin" matches "admin"
// - Wildcard prefix: "admin@*" matches "admin@example.com"
// - Wildcard suffix: "*@example.com" matches "user@example.com"
// - Wildcard middle: "admin@*.com" matches "admin@example.com"
func (rm *RoleMapper) matchPattern(value, pattern string) bool {
	// Exact match
	if value == pattern {
		return true
	}

	// No wildcard in pattern
	if !strings.Contains(pattern, "*") {
		return false
	}

	// Wildcard prefix: "admin@*"
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(value, prefix)
	}

	// Wildcard suffix: "*@example.com"
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(value, suffix)
	}

	// Wildcard middle: "admin@*.com"
	if strings.Count(pattern, "*") == 1 {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(value, parts[0]) && strings.HasSuffix(value, parts[1])
		}
	}

	// Multiple wildcards or complex patterns not supported
	return false
}
