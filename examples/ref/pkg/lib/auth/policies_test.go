package auth

import (
	"reflect"
	"testing"
)

func TestEvaluatePolicy(t *testing.T) {
	policy := ResourcePolicy{
		ID:      "order",
		Type:    "order",
		Version: 1,
		Actions: map[string]PolicyRule{
			"read": {
				AnyOf: []string{"orders:read", "orders:manage"},
			},
			"write": {
				AllOf: []string{"orders:write"},
			},
			"delete": {
				AllOf: []string{"orders:delete", "orders:write"},
			},
			"complex": {
				AllOf: []string{"orders:write"},
				AnyOf: []string{"orders:manage", "orders:admin"},
			},
		},
	}

	tests := []struct {
		name            string
		action          string
		userPermissions []string
		expected        bool
	}{
		{
			name:            "anyOf rule - has first permission",
			action:          "read",
			userPermissions: []string{"orders:read", "users:read"},
			expected:        true,
		},
		{
			name:            "anyOf rule - has second permission",
			action:          "read",
			userPermissions: []string{"orders:manage", "users:read"},
			expected:        true,
		},
		{
			name:            "anyOf rule - has both permissions",
			action:          "read",
			userPermissions: []string{"orders:read", "orders:manage"},
			expected:        true,
		},
		{
			name:            "anyOf rule - has no permissions",
			action:          "read",
			userPermissions: []string{"users:read", "teams:read"},
			expected:        false,
		},
		{
			name:            "allOf rule - has required permission",
			action:          "write",
			userPermissions: []string{"orders:write", "users:read"},
			expected:        true,
		},
		{
			name:            "allOf rule - missing required permission",
			action:          "write",
			userPermissions: []string{"orders:read", "users:read"},
			expected:        false,
		},
		{
			name:            "allOf rule - has all required permissions",
			action:          "delete",
			userPermissions: []string{"orders:delete", "orders:write"},
			expected:        true,
		},
		{
			name:            "allOf rule - missing one required permission",
			action:          "delete",
			userPermissions: []string{"orders:delete"},
			expected:        false,
		},
		{
			name:            "complex rule - has allOf and anyOf",
			action:          "complex",
			userPermissions: []string{"orders:write", "orders:manage"},
			expected:        true,
		},
		{
			name:            "complex rule - has allOf but no anyOf",
			action:          "complex",
			userPermissions: []string{"orders:write"},
			expected:        false,
		},
		{
			name:            "complex rule - has anyOf but no allOf",
			action:          "complex",
			userPermissions: []string{"orders:manage"},
			expected:        false,
		},
		{
			name:            "non-existent action",
			action:          "nonexistent",
			userPermissions: []string{"orders:read"},
			expected:        false,
		},
		{
			name:            "empty user permissions",
			action:          "read",
			userPermissions: []string{},
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvaluatePolicy(policy, tt.action, tt.userPermissions)
			if result != tt.expected {
				t.Errorf("EvaluatePolicy() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluateAllOfRule(t *testing.T) {
	tests := []struct {
		name            string
		allOfRule       []string
		userPermissions []string
		expected        bool
	}{
		{
			name:            "empty allOf rule always passes",
			allOfRule:       []string{},
			userPermissions: []string{"any:permission"},
			expected:        true,
		},
		{
			name:            "single permission required and present",
			allOfRule:       []string{"orders:read"},
			userPermissions: []string{"orders:read", "users:read"},
			expected:        true,
		},
		{
			name:            "single permission required but missing",
			allOfRule:       []string{"orders:read"},
			userPermissions: []string{"users:read"},
			expected:        false,
		},
		{
			name:            "multiple permissions required and all present",
			allOfRule:       []string{"orders:read", "orders:write"},
			userPermissions: []string{"orders:read", "orders:write", "users:read"},
			expected:        true,
		},
		{
			name:            "multiple permissions required but one missing",
			allOfRule:       []string{"orders:read", "orders:write"},
			userPermissions: []string{"orders:read", "users:read"},
			expected:        false,
		},
		{
			name:            "empty user permissions",
			allOfRule:       []string{"orders:read"},
			userPermissions: []string{},
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluateAllOfRule(tt.allOfRule, tt.userPermissions)
			if result != tt.expected {
				t.Errorf("evaluateAllOfRule() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluateAnyOfRule(t *testing.T) {
	tests := []struct {
		name            string
		anyOfRule       []string
		userPermissions []string
		expected        bool
	}{
		{
			name:            "empty anyOf rule always fails",
			anyOfRule:       []string{},
			userPermissions: []string{"any:permission"},
			expected:        false,
		},
		{
			name:            "single permission option and user has it",
			anyOfRule:       []string{"orders:read"},
			userPermissions: []string{"orders:read", "users:read"},
			expected:        true,
		},
		{
			name:            "single permission option and user doesn't have it",
			anyOfRule:       []string{"orders:read"},
			userPermissions: []string{"users:read"},
			expected:        false,
		},
		{
			name:            "multiple options and user has first",
			anyOfRule:       []string{"orders:read", "orders:manage"},
			userPermissions: []string{"orders:read", "users:read"},
			expected:        true,
		},
		{
			name:            "multiple options and user has second",
			anyOfRule:       []string{"orders:read", "orders:manage"},
			userPermissions: []string{"orders:manage", "users:read"},
			expected:        true,
		},
		{
			name:            "multiple options and user has all",
			anyOfRule:       []string{"orders:read", "orders:manage"},
			userPermissions: []string{"orders:read", "orders:manage", "users:read"},
			expected:        true,
		},
		{
			name:            "multiple options and user has none",
			anyOfRule:       []string{"orders:read", "orders:manage"},
			userPermissions: []string{"users:read", "teams:read"},
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluateAnyOfRule(tt.anyOfRule, tt.userPermissions)
			if result != tt.expected {
				t.Errorf("evaluateAnyOfRule() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEvaluateResourceAccess(t *testing.T) {
	policies := []ResourcePolicy{
		{
			ID:      "order",
			Type:    "order",
			Version: 1,
			Actions: map[string]PolicyRule{
				"read": {AnyOf: []string{"orders:read"}},
			},
		},
		{
			ID:      "user",
			Type:    "user",
			Version: 1,
			Actions: map[string]PolicyRule{
				"read": {AnyOf: []string{"users:read"}},
			},
		},
	}

	tests := []struct {
		name            string
		resourceType    string
		action          string
		userPermissions []string
		expected        bool
	}{
		{
			name:            "existing resource and action with permission",
			resourceType:    "order",
			action:          "read",
			userPermissions: []string{"orders:read"},
			expected:        true,
		},
		{
			name:            "existing resource and action without permission",
			resourceType:    "order",
			action:          "read",
			userPermissions: []string{"users:read"},
			expected:        false,
		},
		{
			name:            "non-existent resource type",
			resourceType:    "nonexistent",
			action:          "read",
			userPermissions: []string{"orders:read"},
			expected:        false,
		},
		{
			name:            "existing resource but non-existent action",
			resourceType:    "order",
			action:          "nonexistent",
			userPermissions: []string{"orders:read"},
			expected:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvaluateResourceAccess(policies, tt.resourceType, tt.action, tt.userPermissions)
			if result != tt.expected {
				t.Errorf("EvaluateResourceAccess() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFindPolicy(t *testing.T) {
	policies := []ResourcePolicy{
		{ID: "order", Type: "order", Version: 1},
		{ID: "user", Type: "user", Version: 2},
	}

	tests := []struct {
		name         string
		resourceType string
		expected     *ResourcePolicy
	}{
		{
			name:         "existing resource type",
			resourceType: "order",
			expected:     &policies[0],
		},
		{
			name:         "different existing resource type",
			resourceType: "user",
			expected:     &policies[1],
		},
		{
			name:         "non-existent resource type",
			resourceType: "nonexistent",
			expected:     nil,
		},
		{
			name:         "empty resource type",
			resourceType: "",
			expected:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindPolicy(policies, tt.resourceType)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("FindPolicy() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetLatestPolicyVersion(t *testing.T) {
	policies := []ResourcePolicy{
		{ID: "order-v1", Type: "order", Version: 1},
		{ID: "order-v3", Type: "order", Version: 3},
		{ID: "order-v2", Type: "order", Version: 2},
		{ID: "user-v1", Type: "user", Version: 1},
	}

	tests := []struct {
		name         string
		resourceType string
		expected     *ResourcePolicy
	}{
		{
			name:         "multiple versions returns latest",
			resourceType: "order",
			expected:     &policies[1], // version 3
		},
		{
			name:         "single version returns that version",
			resourceType: "user",
			expected:     &policies[3], // version 1
		},
		{
			name:         "non-existent resource type",
			resourceType: "nonexistent",
			expected:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLatestPolicyVersion(policies, tt.resourceType)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("GetLatestPolicyVersion() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidatePolicy(t *testing.T) {
	tests := []struct {
		name          string
		policy        ResourcePolicy
		expectedCount int
		expectedCodes []string
	}{
		{
			name: "valid policy",
			policy: ResourcePolicy{
				ID:      "order",
				Type:    "order",
				Version: 1,
				Actions: map[string]PolicyRule{
					"read": {AnyOf: []string{"orders:read"}},
				},
			},
			expectedCount: 0,
			expectedCodes: []string{},
		},
		{
			name: "missing ID",
			policy: ResourcePolicy{
				Type:    "order",
				Version: 1,
				Actions: map[string]PolicyRule{
					"read": {AnyOf: []string{"orders:read"}},
				},
			},
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
		{
			name: "missing type",
			policy: ResourcePolicy{
				ID:      "order",
				Version: 1,
				Actions: map[string]PolicyRule{
					"read": {AnyOf: []string{"orders:read"}},
				},
			},
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
		{
			name: "invalid version",
			policy: ResourcePolicy{
				ID:      "order",
				Type:    "order",
				Version: 0,
				Actions: map[string]PolicyRule{
					"read": {AnyOf: []string{"orders:read"}},
				},
			},
			expectedCount: 1,
			expectedCodes: []string{"invalid_value"},
		},
		{
			name: "empty actions",
			policy: ResourcePolicy{
				ID:      "order",
				Type:    "order",
				Version: 1,
				Actions: map[string]PolicyRule{},
			},
			expectedCount: 1,
			expectedCodes: []string{"required"},
		},
		{
			name: "empty rule",
			policy: ResourcePolicy{
				ID:      "order",
				Type:    "order",
				Version: 1,
				Actions: map[string]PolicyRule{
					"read": {AnyOf: []string{}, AllOf: []string{}},
				},
			},
			expectedCount: 1,
			expectedCodes: []string{"empty_rule"},
		},
		{
			name: "invalid permission format",
			policy: ResourcePolicy{
				ID:      "order",
				Type:    "order",
				Version: 1,
				Actions: map[string]PolicyRule{
					"read": {AnyOf: []string{"invalid-permission"}},
				},
			},
			expectedCount: 1,
			expectedCodes: []string{"invalid_format"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := ValidatePolicy(tt.policy)
			if len(errors) != tt.expectedCount {
				t.Errorf("ValidatePolicy() returned %d errors, want %d", len(errors), tt.expectedCount)
			}

			for i, expectedCode := range tt.expectedCodes {
				if i >= len(errors) {
					t.Errorf("Expected error code %s at index %d, but got no error", expectedCode, i)
					continue
				}
				if errors[i].Code != expectedCode {
					t.Errorf("Expected error code %s at index %d, but got %s", expectedCode, i, errors[i].Code)
				}
			}
		})
	}
}

func TestGetPolicyActions(t *testing.T) {
	policy := ResourcePolicy{
		Actions: map[string]PolicyRule{
			"read":   {AnyOf: []string{"orders:read"}},
			"write":  {AnyOf: []string{"orders:write"}},
			"delete": {AnyOf: []string{"orders:delete"}},
		},
	}

	result := GetPolicyActions(policy)
	expected := []string{"read", "write", "delete"}

	if len(result) != len(expected) {
		t.Errorf("GetPolicyActions() returned %d actions, want %d", len(result), len(expected))
		return
	}

	resultMap := make(map[string]bool)
	for _, action := range result {
		resultMap[action] = true
	}

	for _, expectedAction := range expected {
		if !resultMap[expectedAction] {
			t.Errorf("GetPolicyActions() missing expected action: %s", expectedAction)
		}
	}
}

func TestMergePolicies(t *testing.T) {
	basePolicies := []ResourcePolicy{
		{ID: "order-v1", Type: "order", Version: 1},
		{ID: "user-v1", Type: "user", Version: 1},
	}

	overridePolicies := []ResourcePolicy{
		{ID: "order-v2", Type: "order", Version: 2},
		{ID: "team-v1", Type: "team", Version: 1},
	}

	result := MergePolicies(basePolicies, overridePolicies)

	if len(result) != 3 {
		t.Errorf("MergePolicies() returned %d policies, want 3", len(result))
	}

	resultMap := make(map[string]ResourcePolicy)
	for _, policy := range result {
		resultMap[policy.Type] = policy
	}

	if orderPolicy, exists := resultMap["order"]; exists {
		if orderPolicy.Version != 2 {
			t.Errorf("Expected order policy version 2, got %d", orderPolicy.Version)
		}
	} else {
		t.Error("Order policy not found in merged result")
	}

	if _, exists := resultMap["user"]; !exists {
		t.Error("User policy not found in merged result")
	}

	if _, exists := resultMap["team"]; !exists {
		t.Error("Team policy not found in merged result")
	}
}

func TestPolicySupportsAction(t *testing.T) {
	policy := ResourcePolicy{
		Actions: map[string]PolicyRule{
			"read":  {AnyOf: []string{"orders:read"}},
			"write": {AnyOf: []string{"orders:write"}},
		},
	}

	tests := []struct {
		name     string
		action   string
		expected bool
	}{
		{
			name:     "existing action",
			action:   "read",
			expected: true,
		},
		{
			name:     "different existing action",
			action:   "write",
			expected: true,
		},
		{
			name:     "non-existent action",
			action:   "delete",
			expected: false,
		},
		{
			name:     "empty action",
			action:   "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PolicySupportsAction(policy, tt.action)
			if result != tt.expected {
				t.Errorf("PolicySupportsAction() = %v, want %v", result, tt.expected)
			}
		})
	}
}
