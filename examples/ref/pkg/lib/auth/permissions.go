package auth

import "time"

func EvaluatePermissions(grants []Grant, roles []Role, permission string, scope Scope, now time.Time) bool {
	for _, grant := range grants {
		if grant.ExpiresAt != nil && grant.ExpiresAt.Before(now) {
			continue
		}

		if !ScopeMatches(grant.Scope, scope) {
			continue
		}

		if grant.GrantType == GrantTypePermission && grant.Value == permission {
			return true
		}

		if grant.GrantType == GrantTypeRole {
			rolePermissions := GetRolePermissions(roles, grant.Value)
			if ContainsPermission(rolePermissions, permission) {
				return true
			}
		}
	}

	return false
}

func ScopeMatches(grantScope, requestScope Scope) bool {
	if grantScope.Type == "global" {
		return true
	}

	if grantScope.Type != requestScope.Type {
		return false
	}

	return grantScope.ID == requestScope.ID
}

func GetRolePermissions(roles []Role, roleID string) []string {
	for _, role := range roles {
		if role.ID.String() == roleID {
			return role.Permissions
		}
	}
	return nil
}

func ContainsPermission(permissions []string, permission string) bool {
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func FilterValidGrants(grants []Grant, now time.Time) []Grant {
	var validGrants []Grant
	for _, grant := range grants {
		if grant.ExpiresAt == nil || grant.ExpiresAt.After(now) {
			validGrants = append(validGrants, grant)
		}
	}
	return validGrants
}

func GetUserPermissions(grants []Grant, roles []Role, scope Scope, now time.Time) []string {
	var permissions []string
	seen := make(map[string]bool)

	validGrants := FilterValidGrants(grants, now)

	for _, grant := range validGrants {
		if !ScopeMatches(grant.Scope, scope) {
			continue
		}

		if grant.GrantType == GrantTypePermission {
			if !seen[grant.Value] {
				permissions = append(permissions, grant.Value)
				seen[grant.Value] = true
			}
		}

		if grant.GrantType == GrantTypeRole {
			rolePermissions := GetRolePermissions(roles, grant.Value)
			for _, perm := range rolePermissions {
				if !seen[perm] {
					permissions = append(permissions, perm)
					seen[perm] = true
				}
			}
		}
	}

	return permissions
}

func HasAnyPermission(grants []Grant, roles []Role, permissions []string, scope Scope, now time.Time) bool {
	for _, permission := range permissions {
		if EvaluatePermissions(grants, roles, permission, scope, now) {
			return true
		}
	}
	return false
}

func HasAllPermissions(grants []Grant, roles []Role, permissions []string, scope Scope, now time.Time) bool {
	for _, permission := range permissions {
		if !EvaluatePermissions(grants, roles, permission, scope, now) {
			return false
		}
	}
	return true
}

func GetEffectiveScopes(grants []Grant, now time.Time) []Scope {
	var scopes []Scope
	seen := make(map[string]bool)

	validGrants := FilterValidGrants(grants, now)

	for _, grant := range validGrants {
		scopeKey := grant.Scope.Type + ":" + grant.Scope.ID
		if !seen[scopeKey] {
			scopes = append(scopes, grant.Scope)
			seen[scopeKey] = true
		}
	}

	return scopes
}

func IsGlobalAdmin(grants []Grant, roles []Role, now time.Time) bool {
	globalScope := Scope{Type: "global", ID: ""}
	return EvaluatePermissions(grants, roles, "system:admin", globalScope, now)
}

func CanManageRole(grants []Grant, roles []Role, targetRoleID string, scope Scope, now time.Time) bool {
	return EvaluatePermissions(grants, roles, "roles:manage", scope, now)
}

func CanGrantPermission(grants []Grant, roles []Role, targetPermission string, scope Scope, now time.Time) bool {
	manageGrantsPermission := EvaluatePermissions(grants, roles, "grants:manage", scope, now)
	if !manageGrantsPermission {
		return false
	}

	hasTargetPermission := EvaluatePermissions(grants, roles, targetPermission, scope, now)
	return hasTargetPermission
}