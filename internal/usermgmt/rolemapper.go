package usermgmt

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

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
	ctx, span := otel.Tracer("keyline").Start(ctx, "usermgmt.map_groups_to_roles")
	defer span.End()

	span.SetAttributes(
		attribute.StringSlice("groups", groups),
		attribute.Int("groups.count", len(groups)),
	)

	rolesSet := make(map[string]bool)
	matched := false
	matchedPatterns := []string{}

	// Iterate through all role mappings
	for _, mapping := range rm.config.RoleMappings {
		// Match each group against each mapping pattern
		for _, group := range groups {
			if rm.matchPattern(group, mapping.Pattern) {
				matched = true
				matchedPatterns = append(matchedPatterns, mapping.Pattern)

				// Collect ALL matching roles (map deduplicates automatically)
				for _, role := range mapping.ESRoles {
					rolesSet[role] = true
				}

				// Prometheus metrics - record role mapping match
				RoleMappingMatches.WithLabelValues(mapping.Pattern).Inc()

				// Log matched mappings
				slog.InfoContext(ctx, "Role mapping matched",
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
			err := fmt.Errorf("no role mappings matched and no default roles configured")
			span.RecordError(err)
			span.SetStatus(codes.Error, "no roles matched")
			slog.WarnContext(ctx, "Role mapping failed: no matches and no defaults",
				slog.Any("groups", groups),
			)
			return nil, err
		}

		slog.InfoContext(ctx, "No role mappings matched, using default roles",
			slog.Any("default_roles", rm.config.DefaultESRoles),
			slog.Any("groups", groups),
		)

		span.SetAttributes(attribute.Bool("used_defaults", true))
		for _, role := range rm.config.DefaultESRoles {
			rolesSet[role] = true
		}
	} else {
		span.SetAttributes(
			attribute.Bool("used_defaults", false),
			attribute.StringSlice("matched_patterns", matchedPatterns),
		)
	}

	// Convert set to slice
	roles := make([]string, 0, len(rolesSet))
	for role := range rolesSet {
		roles = append(roles, role)
	}

	span.SetAttributes(
		attribute.StringSlice("roles", roles),
		attribute.Int("roles.count", len(roles)),
	)

	slog.InfoContext(ctx, "Role mapping completed",
		slog.Any("groups", groups),
		slog.Any("roles", roles),
		slog.Int("matched_patterns", len(matchedPatterns)),
	)

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
