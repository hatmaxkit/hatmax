package auth

func EvaluatePolicy(policy ResourcePolicy, action string, userPermissions []string) bool {
	rule, exists := policy.Actions[action]
	if !exists {
		return false
	}

	if !evaluateAllOfRule(rule.AllOf, userPermissions) {
		return false
	}

	if len(rule.AnyOf) == 0 {
		return true
	}

	return evaluateAnyOfRule(rule.AnyOf, userPermissions)
}

func evaluateAllOfRule(allOfPermissions []string, userPermissions []string) bool {
	for _, requiredPermission := range allOfPermissions {
		if !containsPermission(userPermissions, requiredPermission) {
			return false
		}
	}
	return true
}

func evaluateAnyOfRule(anyOfPermissions []string, userPermissions []string) bool {
	for _, permission := range anyOfPermissions {
		if containsPermission(userPermissions, permission) {
			return true
		}
	}
	return false
}

func containsPermission(permissions []string, permission string) bool {
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func EvaluateResourceAccess(policies []ResourcePolicy, resourceType, action string, userPermissions []string) bool {
	policy := FindPolicy(policies, resourceType)
	if policy == nil {
		return false
	}

	return EvaluatePolicy(*policy, action, userPermissions)
}

func FindPolicy(policies []ResourcePolicy, resourceType string) *ResourcePolicy {
	for _, policy := range policies {
		if policy.Type == resourceType {
			return &policy
		}
	}
	return nil
}

func GetLatestPolicyVersion(policies []ResourcePolicy, resourceType string) *ResourcePolicy {
	var latestPolicy *ResourcePolicy
	maxVersion := -1

	for _, policy := range policies {
		if policy.Type == resourceType && policy.Version > maxVersion {
			maxVersion = policy.Version
			latestPolicy = &policy
		}
	}

	return latestPolicy
}

func ValidatePolicy(policy ResourcePolicy) ValidationErrors {
	var errors ValidationErrors

	if policy.ID == "" {
		errors = append(errors, ValidationError{
			Field:   "id",
			Code:    "required",
			Message: "Policy ID is required",
		})
	}

	if policy.Type == "" {
		errors = append(errors, ValidationError{
			Field:   "type",
			Code:    "required",
			Message: "Policy type is required",
		})
	}

	if policy.Version < 1 {
		errors = append(errors, ValidationError{
			Field:   "version",
			Code:    "invalid_value",
			Message: "Policy version must be greater than 0",
		})
	}

	if len(policy.Actions) == 0 {
		errors = append(errors, ValidationError{
			Field:   "actions",
			Code:    "required",
			Message: "Policy must define at least one action",
		})
	}

	for actionName, rule := range policy.Actions {
		if actionName == "" {
			errors = append(errors, ValidationError{
				Field:   "actions",
				Code:    "empty_action_name",
				Message: "Action name cannot be empty",
			})
			continue
		}

		ruleErrors := ValidatePolicyRule(rule, "actions."+actionName)
		errors = append(errors, ruleErrors...)
	}

	return errors
}

func ValidatePolicyRule(rule PolicyRule, fieldPrefix string) ValidationErrors {
	var errors ValidationErrors

	if len(rule.AllOf) == 0 && len(rule.AnyOf) == 0 {
		errors = append(errors, ValidationError{
			Field:   fieldPrefix,
			Code:    "empty_rule",
			Message: "Policy rule must define at least one permission in allOf or anyOf",
		})
	}

	for i, permission := range rule.AllOf {
		permErrors := ValidatePermissionCode(permission)
		for _, err := range permErrors {
			errors = append(errors, ValidationError{
				Field:   fieldPrefix + ".allOf[" + string(rune(i)) + "]",
				Code:    err.Code,
				Message: err.Message,
			})
		}
	}

	for i, permission := range rule.AnyOf {
		permErrors := ValidatePermissionCode(permission)
		for _, err := range permErrors {
			errors = append(errors, ValidationError{
				Field:   fieldPrefix + ".anyOf[" + string(rune(i)) + "]",
				Code:    err.Code,
				Message: err.Message,
			})
		}
	}

	return errors
}

func GetPolicyActions(policy ResourcePolicy) []string {
	actions := make([]string, 0, len(policy.Actions))
	for action := range policy.Actions {
		actions = append(actions, action)
	}
	return actions
}

func GetRequiredPermissions(policy ResourcePolicy, action string) []string {
	rule, exists := policy.Actions[action]
	if !exists {
		return nil
	}

	var permissions []string
	seen := make(map[string]bool)

	for _, perm := range rule.AllOf {
		if !seen[perm] {
			permissions = append(permissions, perm)
			seen[perm] = true
		}
	}

	for _, perm := range rule.AnyOf {
		if !seen[perm] {
			permissions = append(permissions, perm)
			seen[perm] = true
		}
	}

	return permissions
}

func MergePolicies(basePolicies, overridePolicies []ResourcePolicy) []ResourcePolicy {
	merged := make(map[string]ResourcePolicy)

	for _, policy := range basePolicies {
		merged[policy.Type] = policy
	}

	for _, policy := range overridePolicies {
		if existing, exists := merged[policy.Type]; exists {
			if policy.Version >= existing.Version {
				merged[policy.Type] = policy
			}
		} else {
			merged[policy.Type] = policy
		}
	}

	result := make([]ResourcePolicy, 0, len(merged))
	for _, policy := range merged {
		result = append(result, policy)
	}

	return result
}

func PolicySupportsAction(policy ResourcePolicy, action string) bool {
	_, exists := policy.Actions[action]
	return exists
}

func GetAllPolicyPermissions(policy ResourcePolicy) []string {
	var permissions []string
	seen := make(map[string]bool)

	for _, rule := range policy.Actions {
		for _, perm := range rule.AllOf {
			if !seen[perm] {
				permissions = append(permissions, perm)
				seen[perm] = true
			}
		}
		for _, perm := range rule.AnyOf {
			if !seen[perm] {
				permissions = append(permissions, perm)
				seen[perm] = true
			}
		}
	}

	return permissions
}