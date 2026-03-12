package usermgmt

import (
	"context"
	"sort"
	"testing"

	"github.com/yourusername/keyline/internal/config"
)

// Helper function to create a test config with role mappings
func createTestConfig(mappings []config.RoleMapping, defaultRoles []string) *config.Config {
	return &config.Config{
		RoleMappings:   mappings,
		DefaultESRoles: defaultRoles,
	}
}

// Helper function to sort roles for comparison
func sortRoles(roles []string) []string {
	sorted := make([]string, len(roles))
	copy(sorted, roles)
	sort.Strings(sorted)
	return sorted
}

// Helper function to compare role slices (order-independent)
func rolesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sortedA := sortRoles(a)
	sortedB := sortRoles(b)
	for i := range sortedA {
		if sortedA[i] != sortedB[i] {
			return false
		}
	}
	return true
}

// TestMatchPattern_ExactMatch tests exact pattern matching
func TestMatchPattern_ExactMatch(t *testing.T) {
	rm := NewRoleMapper(&config.Config{})

	tests := []struct {
		name    string
		value   string
		pattern string
		want    bool
	}{
		{
			name:    "exact match - simple",
			value:   "admin",
			pattern: "admin",
			want:    true,
		},
		{
			name:    "exact match - with special chars",
			value:   "admin@example.com",
			pattern: "admin@example.com",
			want:    true,
		},
		{
			name:    "no match - different value",
			value:   "user",
			pattern: "admin",
			want:    false,
		},
		{
			name:    "no match - case sensitive",
			value:   "Admin",
			pattern: "admin",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rm.matchPattern(tt.value, tt.pattern)
			if got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.value, tt.pattern, got, tt.want)
			}
		})
	}
}

// TestMatchPattern_WildcardPrefix tests wildcard prefix matching (pattern ends with *)
func TestMatchPattern_WildcardPrefix(t *testing.T) {
	rm := NewRoleMapper(&config.Config{})

	tests := []struct {
		name    string
		value   string
		pattern string
		want    bool
	}{
		{
			name:    "wildcard prefix - matches",
			value:   "admin@example.com",
			pattern: "admin@*",
			want:    true,
		},
		{
			name:    "wildcard prefix - matches with subdomain",
			value:   "admin@mail.example.com",
			pattern: "admin@*",
			want:    true,
		},
		{
			name:    "wildcard prefix - no match",
			value:   "user@example.com",
			pattern: "admin@*",
			want:    false,
		},
		{
			name:    "wildcard prefix - empty after prefix",
			value:   "admin@",
			pattern: "admin@*",
			want:    true,
		},
		{
			name:    "wildcard prefix - exact prefix",
			value:   "admin",
			pattern: "admin*",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rm.matchPattern(tt.value, tt.pattern)
			if got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.value, tt.pattern, got, tt.want)
			}
		})
	}
}

// TestMatchPattern_WildcardSuffix tests wildcard suffix matching (pattern starts with *)
func TestMatchPattern_WildcardSuffix(t *testing.T) {
	rm := NewRoleMapper(&config.Config{})

	tests := []struct {
		name    string
		value   string
		pattern string
		want    bool
	}{
		{
			name:    "wildcard suffix - matches",
			value:   "user@example.com",
			pattern: "*@example.com",
			want:    true,
		},
		{
			name:    "wildcard suffix - matches different user",
			value:   "admin@example.com",
			pattern: "*@example.com",
			want:    true,
		},
		{
			name:    "wildcard suffix - no match",
			value:   "user@other.com",
			pattern: "*@example.com",
			want:    false,
		},
		{
			name:    "wildcard suffix - empty before suffix",
			value:   "@example.com",
			pattern: "*@example.com",
			want:    true,
		},
		{
			name:    "wildcard suffix - partial match not enough",
			value:   "user@example.com.au",
			pattern: "*@example.com",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rm.matchPattern(tt.value, tt.pattern)
			if got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.value, tt.pattern, got, tt.want)
			}
		})
	}
}

// TestMatchPattern_WildcardMiddle tests wildcard middle matching (pattern has * in middle)
func TestMatchPattern_WildcardMiddle(t *testing.T) {
	rm := NewRoleMapper(&config.Config{})

	tests := []struct {
		name    string
		value   string
		pattern string
		want    bool
	}{
		{
			name:    "wildcard middle - matches",
			value:   "admin@example.com",
			pattern: "admin@*.com",
			want:    true,
		},
		{
			name:    "wildcard middle - matches with subdomain",
			value:   "admin@mail.example.com",
			pattern: "admin@*.com",
			want:    true,
		},
		{
			name:    "wildcard middle - no match prefix",
			value:   "user@example.com",
			pattern: "admin@*.com",
			want:    false,
		},
		{
			name:    "wildcard middle - no match suffix",
			value:   "admin@example.org",
			pattern: "admin@*.com",
			want:    false,
		},
		{
			name:    "wildcard middle - empty middle",
			value:   "admin@.com",
			pattern: "admin@*.com",
			want:    true,
		},
		{
			name:    "wildcard middle - complex pattern",
			value:   "dev-team-lead",
			pattern: "dev-*-lead",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rm.matchPattern(tt.value, tt.pattern)
			if got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.value, tt.pattern, got, tt.want)
			}
		})
	}
}

// TestMatchPattern_NoMatch tests cases where no match should occur
func TestMatchPattern_NoMatch(t *testing.T) {
	rm := NewRoleMapper(&config.Config{})

	tests := []struct {
		name    string
		value   string
		pattern string
		want    bool
	}{
		{
			name:    "no match - completely different",
			value:   "user",
			pattern: "admin",
			want:    false,
		},
		{
			name:    "no match - partial overlap",
			value:   "administrator",
			pattern: "admin",
			want:    false,
		},
		{
			name:    "no match - empty value",
			value:   "",
			pattern: "admin",
			want:    false,
		},
		{
			name:    "no match - empty pattern",
			value:   "admin",
			pattern: "",
			want:    false,
		},
		{
			name:    "no match - multiple wildcards not supported",
			value:   "admin@example.com",
			pattern: "*@*.*",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rm.matchPattern(tt.value, tt.pattern)
			if got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.value, tt.pattern, got, tt.want)
			}
		})
	}
}

// TestMapGroupsToRoles_SingleGroupSingleRole tests mapping a single group to a single role
func TestMapGroupsToRoles_SingleGroupSingleRole(t *testing.T) {
	cfg := createTestConfig(
		[]config.RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
		},
		nil,
	)

	rm := NewRoleMapper(cfg)
	ctx := context.Background()

	roles, err := rm.MapGroupsToRoles(ctx, []string{"admin"})
	if err != nil {
		t.Fatalf("MapGroupsToRoles() error = %v", err)
	}

	expected := []string{"superuser"}
	if !rolesEqual(roles, expected) {
		t.Errorf("MapGroupsToRoles() = %v, want %v", roles, expected)
	}
}

// TestMapGroupsToRoles_SingleGroupMultipleRoles tests mapping a single group to multiple roles
func TestMapGroupsToRoles_SingleGroupMultipleRoles(t *testing.T) {
	cfg := createTestConfig(
		[]config.RoleMapping{
			{
				Claim:   "groups",
				Pattern: "developers",
				ESRoles: []string{"developer", "kibana_user"},
			},
		},
		nil,
	)

	rm := NewRoleMapper(cfg)
	ctx := context.Background()

	roles, err := rm.MapGroupsToRoles(ctx, []string{"developers"})
	if err != nil {
		t.Fatalf("MapGroupsToRoles() error = %v", err)
	}

	expected := []string{"developer", "kibana_user"}
	if !rolesEqual(roles, expected) {
		t.Errorf("MapGroupsToRoles() = %v, want %v", roles, expected)
	}
}

// TestMapGroupsToRoles_MultipleGroupsMultipleRoles tests accumulation of roles from multiple groups
func TestMapGroupsToRoles_MultipleGroupsMultipleRoles(t *testing.T) {
	cfg := createTestConfig(
		[]config.RoleMapping{
			{
				Claim:   "groups",
				Pattern: "developers",
				ESRoles: []string{"developer", "kibana_user"},
			},
			{
				Claim:   "groups",
				Pattern: "users",
				ESRoles: []string{"viewer"},
			},
		},
		nil,
	)

	rm := NewRoleMapper(cfg)
	ctx := context.Background()

	roles, err := rm.MapGroupsToRoles(ctx, []string{"developers", "users"})
	if err != nil {
		t.Fatalf("MapGroupsToRoles() error = %v", err)
	}

	expected := []string{"developer", "kibana_user", "viewer"}
	if !rolesEqual(roles, expected) {
		t.Errorf("MapGroupsToRoles() = %v, want %v", roles, expected)
	}
}

// TestMapGroupsToRoles_NoMatchesUseDefaultRoles tests using default roles when no mappings match
func TestMapGroupsToRoles_NoMatchesUseDefaultRoles(t *testing.T) {
	cfg := createTestConfig(
		[]config.RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
		},
		[]string{"viewer", "kibana_user"},
	)

	rm := NewRoleMapper(cfg)
	ctx := context.Background()

	// Groups that don't match any pattern
	roles, err := rm.MapGroupsToRoles(ctx, []string{"unknown-group"})
	if err != nil {
		t.Fatalf("MapGroupsToRoles() error = %v", err)
	}

	expected := []string{"viewer", "kibana_user"}
	if !rolesEqual(roles, expected) {
		t.Errorf("MapGroupsToRoles() = %v, want %v", roles, expected)
	}
}

// TestMapGroupsToRoles_NoMatchesNoDefaults tests error when no mappings match and no defaults
func TestMapGroupsToRoles_NoMatchesNoDefaults(t *testing.T) {
	cfg := createTestConfig(
		[]config.RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
		},
		nil, // No default roles
	)

	rm := NewRoleMapper(cfg)
	ctx := context.Background()

	// Groups that don't match any pattern
	_, err := rm.MapGroupsToRoles(ctx, []string{"unknown-group"})
	if err == nil {
		t.Fatal("MapGroupsToRoles() expected error, got nil")
	}

	expectedError := "no role mappings matched and no default roles configured"
	if err.Error() != expectedError {
		t.Errorf("MapGroupsToRoles() error = %v, want %v", err.Error(), expectedError)
	}
}

// TestMapGroupsToRoles_EmptyGroups tests behavior with empty groups array
func TestMapGroupsToRoles_EmptyGroups(t *testing.T) {
	cfg := createTestConfig(
		[]config.RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
		},
		[]string{"viewer"},
	)

	rm := NewRoleMapper(cfg)
	ctx := context.Background()

	// Empty groups should use default roles
	roles, err := rm.MapGroupsToRoles(ctx, []string{})
	if err != nil {
		t.Fatalf("MapGroupsToRoles() error = %v", err)
	}

	expected := []string{"viewer"}
	if !rolesEqual(roles, expected) {
		t.Errorf("MapGroupsToRoles() = %v, want %v", roles, expected)
	}
}

// TestMapGroupsToRoles_RoleDeduplication tests that duplicate roles are deduplicated
func TestMapGroupsToRoles_RoleDeduplication(t *testing.T) {
	cfg := createTestConfig(
		[]config.RoleMapping{
			{
				Claim:   "groups",
				Pattern: "developers",
				ESRoles: []string{"developer", "kibana_user"},
			},
			{
				Claim:   "groups",
				Pattern: "frontend-developers",
				ESRoles: []string{"developer", "viewer"}, // "developer" is duplicate
			},
		},
		nil,
	)

	rm := NewRoleMapper(cfg)
	ctx := context.Background()

	roles, err := rm.MapGroupsToRoles(ctx, []string{"developers", "frontend-developers"})
	if err != nil {
		t.Fatalf("MapGroupsToRoles() error = %v", err)
	}

	// Should have 3 unique roles, not 4
	expected := []string{"developer", "kibana_user", "viewer"}
	if !rolesEqual(roles, expected) {
		t.Errorf("MapGroupsToRoles() = %v, want %v", roles, expected)
	}

	// Verify no duplicates
	roleSet := make(map[string]bool)
	for _, role := range roles {
		if roleSet[role] {
			t.Errorf("MapGroupsToRoles() returned duplicate role: %s", role)
		}
		roleSet[role] = true
	}
}

// TestMapGroupsToRoles_WildcardPatterns tests role mapping with wildcard patterns
func TestMapGroupsToRoles_WildcardPatterns(t *testing.T) {
	cfg := createTestConfig(
		[]config.RoleMapping{
			{
				Claim:   "groups",
				Pattern: "*-admins",
				ESRoles: []string{"superuser"},
			},
			{
				Claim:   "groups",
				Pattern: "*@example.com",
				ESRoles: []string{"viewer"},
			},
		},
		nil,
	)

	rm := NewRoleMapper(cfg)
	ctx := context.Background()

	tests := []struct {
		name     string
		groups   []string
		expected []string
	}{
		{
			name:     "wildcard prefix match",
			groups:   []string{"system-admins"},
			expected: []string{"superuser"},
		},
		{
			name:     "wildcard suffix match",
			groups:   []string{"user@example.com"},
			expected: []string{"viewer"},
		},
		{
			name:     "multiple wildcard matches",
			groups:   []string{"db-admins", "admin@example.com"},
			expected: []string{"superuser", "viewer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles, err := rm.MapGroupsToRoles(ctx, tt.groups)
			if err != nil {
				t.Fatalf("MapGroupsToRoles() error = %v", err)
			}

			if !rolesEqual(roles, tt.expected) {
				t.Errorf("MapGroupsToRoles() = %v, want %v", roles, tt.expected)
			}
		})
	}
}

// TestMapGroupsToRoles_ComplexScenario tests a complex real-world scenario
func TestMapGroupsToRoles_ComplexScenario(t *testing.T) {
	cfg := createTestConfig(
		[]config.RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
			{
				Claim:   "groups",
				Pattern: "superusers",
				ESRoles: []string{"superuser"},
			},
			{
				Claim:   "groups",
				Pattern: "*-developers",
				ESRoles: []string{"developer", "kibana_user"},
			},
			{
				Claim:   "groups",
				Pattern: "developers",
				ESRoles: []string{"developer", "kibana_user"},
			},
			{
				Claim:   "groups",
				Pattern: "viewers",
				ESRoles: []string{"viewer"},
			},
			{
				Claim:   "email",
				Pattern: "*@admin.example.com",
				ESRoles: []string{"superuser"},
			},
		},
		[]string{"viewer", "kibana_user"},
	)

	rm := NewRoleMapper(cfg)
	ctx := context.Background()

	tests := []struct {
		name     string
		groups   []string
		expected []string
		wantErr  bool
	}{
		{
			name:     "admin user",
			groups:   []string{"admin"},
			expected: []string{"superuser"},
			wantErr:  false,
		},
		{
			name:     "developer with multiple groups",
			groups:   []string{"developers", "users"},
			expected: []string{"developer", "kibana_user"},
			wantErr:  false,
		},
		{
			name:     "wildcard developer match",
			groups:   []string{"frontend-developers"},
			expected: []string{"developer", "kibana_user"},
			wantErr:  false,
		},
		{
			name:     "no match uses defaults",
			groups:   []string{"unknown-group"},
			expected: []string{"viewer", "kibana_user"},
			wantErr:  false,
		},
		{
			name:     "multiple matches with deduplication",
			groups:   []string{"admin", "superusers"},
			expected: []string{"superuser"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles, err := rm.MapGroupsToRoles(ctx, tt.groups)
			if (err != nil) != tt.wantErr {
				t.Fatalf("MapGroupsToRoles() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !rolesEqual(roles, tt.expected) {
				t.Errorf("MapGroupsToRoles() = %v, want %v", roles, tt.expected)
			}
		})
	}
}

// Property-Based Tests

// TestProperty_MappingIsDeterministic tests that mapping is deterministic
// Property: Same groups → same roles (order-independent)
func TestProperty_MappingIsDeterministic(t *testing.T) {
	cfg := createTestConfig(
		[]config.RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
			{
				Claim:   "groups",
				Pattern: "developers",
				ESRoles: []string{"developer", "kibana_user"},
			},
			{
				Claim:   "groups",
				Pattern: "*-admins",
				ESRoles: []string{"superuser"},
			},
		},
		[]string{"viewer"},
	)

	rm := NewRoleMapper(cfg)
	ctx := context.Background()

	// Test cases with various group combinations
	testCases := [][]string{
		{"admin"},
		{"developers"},
		{"admin", "developers"},
		{"developers", "admin"}, // Different order
		{"system-admins"},
		{"unknown-group"},
		{},
		{"admin", "developers", "system-admins"},
		{"system-admins", "developers", "admin"}, // Different order
	}

	for _, groups := range testCases {
		// Run mapping multiple times
		var results [][]string
		for i := 0; i < 10; i++ {
			roles, err := rm.MapGroupsToRoles(ctx, groups)
			if err != nil {
				// If error occurs, it should be consistent
				for j := 0; j < 5; j++ {
					_, err2 := rm.MapGroupsToRoles(ctx, groups)
					if err2 == nil {
						t.Errorf("Property violation: mapping is not deterministic for groups %v (error inconsistency)", groups)
					}
				}
				continue
			}
			results = append(results, roles)
		}

		// Verify all results are identical (order-independent)
		if len(results) > 0 {
			first := results[0]
			for i, result := range results {
				if !rolesEqual(first, result) {
					t.Errorf("Property violation: mapping is not deterministic for groups %v\n  Run 0: %v\n  Run %d: %v",
						groups, first, i, result)
				}
			}
		}
	}
}

// TestProperty_MultipleGroupsAccumulateRoles tests that multiple groups accumulate roles
// Property: Multiple groups accumulate roles (no duplicates)
func TestProperty_MultipleGroupsAccumulateRoles(t *testing.T) {
	cfg := createTestConfig(
		[]config.RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
			{
				Claim:   "groups",
				Pattern: "developers",
				ESRoles: []string{"developer", "kibana_user"},
			},
			{
				Claim:   "groups",
				Pattern: "viewers",
				ESRoles: []string{"viewer"},
			},
			{
				Claim:   "groups",
				Pattern: "users",
				ESRoles: []string{"kibana_user"}, // Overlaps with developers
			},
		},
		nil,
	)

	rm := NewRoleMapper(cfg)
	ctx := context.Background()

	// Property: roles(A + B) should contain all roles from roles(A) and roles(B)
	testCases := []struct {
		groupsA  []string
		groupsB  []string
		combined []string
	}{
		{
			groupsA:  []string{"admin"},
			groupsB:  []string{"developers"},
			combined: []string{"admin", "developers"},
		},
		{
			groupsA:  []string{"developers"},
			groupsB:  []string{"viewers"},
			combined: []string{"developers", "viewers"},
		},
		{
			groupsA:  []string{"developers"},
			groupsB:  []string{"users"},
			combined: []string{"developers", "users"},
		},
	}

	for _, tc := range testCases {
		rolesA, errA := rm.MapGroupsToRoles(ctx, tc.groupsA)
		rolesB, errB := rm.MapGroupsToRoles(ctx, tc.groupsB)
		rolesCombined, errCombined := rm.MapGroupsToRoles(ctx, tc.combined)

		// Skip if any mapping failed
		if errA != nil || errB != nil || errCombined != nil {
			continue
		}

		// Create set of expected roles (union of A and B)
		expectedSet := make(map[string]bool)
		for _, role := range rolesA {
			expectedSet[role] = true
		}
		for _, role := range rolesB {
			expectedSet[role] = true
		}

		// Verify combined roles contain all expected roles
		combinedSet := make(map[string]bool)
		for _, role := range rolesCombined {
			combinedSet[role] = true
		}

		for expectedRole := range expectedSet {
			if !combinedSet[expectedRole] {
				t.Errorf("Property violation: combined groups %v missing role %s\n  Groups A %v → %v\n  Groups B %v → %v\n  Combined %v → %v",
					tc.combined, expectedRole,
					tc.groupsA, rolesA,
					tc.groupsB, rolesB,
					tc.combined, rolesCombined)
			}
		}

		// Verify no duplicates in combined result
		if len(rolesCombined) != len(combinedSet) {
			t.Errorf("Property violation: combined result has duplicates for groups %v: %v",
				tc.combined, rolesCombined)
		}
	}
}

// TestProperty_RoleDeduplication tests that roles are always deduplicated
// Property: No duplicate roles in result
func TestProperty_RoleDeduplication(t *testing.T) {
	cfg := createTestConfig(
		[]config.RoleMapping{
			{
				Claim:   "groups",
				Pattern: "developers",
				ESRoles: []string{"developer", "kibana_user"},
			},
			{
				Claim:   "groups",
				Pattern: "frontend-developers",
				ESRoles: []string{"developer", "viewer"}, // "developer" overlaps
			},
			{
				Claim:   "groups",
				Pattern: "backend-developers",
				ESRoles: []string{"developer", "admin"}, // "developer" overlaps
			},
			{
				Claim:   "groups",
				Pattern: "users",
				ESRoles: []string{"kibana_user", "viewer"}, // Multiple overlaps
			},
		},
		nil,
	)

	rm := NewRoleMapper(cfg)
	ctx := context.Background()

	// Test various combinations that should produce overlapping roles
	testCases := [][]string{
		{"developers", "frontend-developers"},
		{"developers", "backend-developers"},
		{"frontend-developers", "backend-developers"},
		{"developers", "frontend-developers", "backend-developers"},
		{"developers", "users"},
		{"frontend-developers", "users"},
		{"developers", "frontend-developers", "backend-developers", "users"},
	}

	for _, groups := range testCases {
		roles, err := rm.MapGroupsToRoles(ctx, groups)
		if err != nil {
			continue
		}

		// Property: No duplicates
		roleSet := make(map[string]bool)
		for _, role := range roles {
			if roleSet[role] {
				t.Errorf("Property violation: duplicate role %s in result for groups %v: %v",
					role, groups, roles)
			}
			roleSet[role] = true
		}

		// Property: Length of result equals number of unique roles
		if len(roles) != len(roleSet) {
			t.Errorf("Property violation: result length mismatch for groups %v\n  Result: %v\n  Length: %d, Unique: %d",
				groups, roles, len(roles), len(roleSet))
		}
	}
}

// TestProperty_DefaultRolesOnlyWhenNoMatch tests that default roles are only used when no mappings match
// Property: Default roles used if and only if no mappings matched
func TestProperty_DefaultRolesOnlyWhenNoMatch(t *testing.T) {
	cfg := createTestConfig(
		[]config.RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
			{
				Claim:   "groups",
				Pattern: "developers",
				ESRoles: []string{"developer"},
			},
		},
		[]string{"default_viewer", "default_user"},
	)

	rm := NewRoleMapper(cfg)
	ctx := context.Background()

	testCases := []struct {
		groups            []string
		shouldUseDefaults bool
	}{
		{
			groups:            []string{"admin"},
			shouldUseDefaults: false, // Matches "admin" pattern
		},
		{
			groups:            []string{"developers"},
			shouldUseDefaults: false, // Matches "developers" pattern
		},
		{
			groups:            []string{"unknown-group"},
			shouldUseDefaults: true, // No match
		},
		{
			groups:            []string{},
			shouldUseDefaults: true, // Empty groups, no match
		},
		{
			groups:            []string{"admin", "unknown-group"},
			shouldUseDefaults: false, // At least one match
		},
	}

	for _, tc := range testCases {
		roles, err := rm.MapGroupsToRoles(ctx, tc.groups)
		if err != nil {
			t.Fatalf("MapGroupsToRoles(%v) error = %v", tc.groups, err)
		}

		// Check if result contains default roles
		hasDefaultRoles := false
		for _, role := range roles {
			if role == "default_viewer" || role == "default_user" {
				hasDefaultRoles = true
				break
			}
		}

		if hasDefaultRoles != tc.shouldUseDefaults {
			t.Errorf("Property violation for groups %v:\n  Expected default roles: %v\n  Got default roles: %v\n  Result: %v",
				tc.groups, tc.shouldUseDefaults, hasDefaultRoles, roles)
		}

		// If using defaults, result should ONLY contain default roles
		if tc.shouldUseDefaults {
			expected := []string{"default_viewer", "default_user"}
			if !rolesEqual(roles, expected) {
				t.Errorf("Property violation: when using defaults, result should only contain default roles\n  Groups: %v\n  Expected: %v\n  Got: %v",
					tc.groups, expected, roles)
			}
		}
	}
}

// TestProperty_EmptyRolesNeverReturned tests that empty role list is never returned on success
// Property: Successful mapping always returns at least one role
func TestProperty_EmptyRolesNeverReturned(t *testing.T) {
	cfg := createTestConfig(
		[]config.RoleMapping{
			{
				Claim:   "groups",
				Pattern: "admin",
				ESRoles: []string{"superuser"},
			},
			{
				Claim:   "groups",
				Pattern: "developers",
				ESRoles: []string{"developer", "kibana_user"},
			},
		},
		[]string{"viewer"},
	)

	rm := NewRoleMapper(cfg)
	ctx := context.Background()

	testCases := [][]string{
		{"admin"},
		{"developers"},
		{"admin", "developers"},
		{"unknown-group"}, // Should use defaults
		{},                // Empty groups, should use defaults
		{"admin", "unknown-group"},
	}

	for _, groups := range testCases {
		roles, err := rm.MapGroupsToRoles(ctx, groups)
		if err != nil {
			// Error is acceptable, but if no error, must have roles
			continue
		}

		// Property: If no error, must have at least one role
		if len(roles) == 0 {
			t.Errorf("Property violation: successful mapping returned empty roles for groups %v", groups)
		}
	}
}
